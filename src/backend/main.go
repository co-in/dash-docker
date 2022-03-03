package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

var f *os.File
var dashCLILocation = "/usr/local/bin/dash-cli"
var dashDaemonLocation = "/usr/local/bin/dashd"
var dashLogLocation = "/dashd/.dashcore/"
var isFirstCheck = false
var (
	debugMode = flag.Bool("debug", true, "Debug mode")
	txId      = flag.String("tx", "", "Transaction ID")
	wsPort    = flag.Int("port", 8080, "Websocket port")
)

func CLI(args ...string) (string, bool) {
	cmd := exec.Command(dashCLILocation, args...)
	stdout, err := cmd.Output()
	if err != nil {
		switch errType := err.(type) {
		case *exec.ExitError:
			return string(errType.Stderr), false
		case *exec.Error:
			return errType.Err.Error(), false
		default:
			return "unexpected error", false
		}
	}

	return strings.Trim(string(stdout), "\n \t"), true
}

func StartLog() {
	if !*debugMode {
		return
	}

	var err error
	f, err = os.OpenFile(dashLogLocation+"dash-instant-notify.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
}

func FinishLog() {
	if f != nil {
		_ = f.Close()
	}
}

var sockets = make(map[string]*websocket.Conn)

type RawTx struct {
	Vout []struct {
		ValueSat     int64
		ScriptPubKey struct {
			Addresses []string
		}
	}
}
type RawSync struct {
	IsBlockchainSynced bool
	IsSynced           bool
}
type RawInfo struct {
	Headers int
	Blocks  int
}

type Response struct {
	Type  string `json:"type"`
	Error string `json:"error"`
	Value string `json:"value"`
}

var syncInfo RawInfo

//Triggered after daemon receive incoming transaction
func processTransaction(txId string) {
	var err error

	log.Println("recv", txId, sockets)

	var tx RawTx
	resp, ok := CLI("getrawtransaction", txId, "true")
	if !ok {
		log.Println("err1", err)
		return
	}
	err = json.Unmarshal([]byte(resp), &tx)
	if err != nil {
		log.Println("err2", err)
		return
	}

	var balance int64
	var address string
	var addresses = make(map[string]int64)

	for _, out := range tx.Vout {
		for _, address = range out.ScriptPubKey.Addresses {
			_, ok = sockets[address]

			if !ok {
				continue
			}

			//In one transaction address can be duplicate
			if _, ok = addresses[address]; ok {
				continue
			}

			////Get balance for each addresses
			//balance, ok = CLI("getaddressbalance", fmt.Sprintf(`{"addresses":["%s"]}`, address))
			//log.Println("DEBUG2", balance)
			//
			//if !ok {
			//	log.Println("err3", balance)
			//	return
			//}
			//addresses[address] = balance

			addresses[address] += out.ValueSat
		}
	}

	var data []byte
	for address, balance = range addresses {
		log.Println("DEBUG_BALANCE", balance)
		data, err = json.Marshal(&Response{
			Type:  "tx",
			Value: fmt.Sprintf("%0.8f", float64(balance)/100000000),
		})
		if err != nil {
			log.Println("err4", err)
			continue
		}

		_, err = sockets[address].Write(data)
		if err != nil {
			log.Println("err5", err)
			continue
		}
	}

	log.Println("done", string(data), addresses, sockets)

	return
}

func processWebSocket() {
	var err error

	portStr := fmt.Sprintf(":%d", *wsPort)
	fmt.Printf("Start WebSocket at port %s\n", portStr)
	fmt.Printf("If your run locally visit  http://127.0.0.1%s/\n", portStr)

	http.Handle("/trigger", websocket.Handler(func(ws *websocket.Conn) {
		var buff = make([]byte, 1024)
		n, err := ws.Read(buff)
		if err != nil {
			log.Println("n-err-16", err)
			return
		}
		log.Println("ValueRaw", buff[:n])
		r := new(Response)
		err = json.Unmarshal(buff[:n], r)
		if err != nil {
			log.Println("n-err-17", err)
			return
		}
		log.Println("Value", r.Value)
		processTransaction(r.Value)
	}))
	http.Handle("/notify", websocket.Handler(func(ws *websocket.Conn) {
		addressOrErr, ok := CLI("getnewaddress")
		var data []byte

		if !ok {
			log.Println("n-err-1", err)
			data, err = json.Marshal(&Response{
				Type:  "sign-up",
				Error: addressOrErr,
			})
			if err != nil {
				log.Println("n-err-2", err)
				return
			}
			_, err = ws.Write(data)
			if err != nil {
				log.Println("n-err-3", err)
				return
			}
		}

		sockets[addressOrErr] = ws

		log.Println("add", addressOrErr, sockets)

		//The socket is closed after read from him
		defer func() {
			log.Println("delete", addressOrErr)
			delete(sockets, addressOrErr)
		}()

		data, err = json.Marshal(&Response{
			Type:  "sign-up",
			Value: addressOrErr,
		})
		if err != nil {
			log.Println("n-err-4", err)
			return
		}
		_, err = ws.Write(data)
		if err != nil {
			log.Println("n-err-5", err)
			return
		}

		//Wait from incoming socket message
		var buff = make([]byte, 1024)
		_, err = ws.Read(buff)
		if err != nil {
			//log.Println("n-err-6", err)
			return
		}
	}))

	err = http.ListenAndServe(portStr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: " + err.Error())
	}
}

func getBlockCount() string {
	bcs, ok := CLI("getblockcount")
	if !ok {
		return "#"
	}

	bc, err := strconv.Atoi(bcs)
	if err != nil {
		return "#"
	}

	bcs, ok = CLI("getblockchaininfo")
	if !ok {
		return "#"
	}

	err = json.Unmarshal([]byte(bcs), &syncInfo)
	if err != nil {
		return "#"
	}

	return fmt.Sprintf("\rStat: [headers:%d, downloaded:%d, validated: %d]", syncInfo.Headers, syncInfo.Blocks, bc)
}

func WaitSync() {
	var ok bool
	var bcs, bci string
	var err error
	var info RawSync

	for {
		bcs, ok = CLI("getblockcount")
		if !ok {
			if !isFirstCheck {
				isFirstCheck = true
				fmt.Println("Wait Daemon ...")
			}

			time.Sleep(5 * time.Second)

			continue
		}

		_, err = strconv.Atoi(bcs)
		if err == nil {
			break
		}
	}

	isFirstCheck = false

	for {
		bci, ok = CLI("mnsync", "status")
		if !ok {
			ShowFirstCheck()
			continue
		}

		if json.Unmarshal([]byte(bci), &info) != nil {
			ShowFirstCheck()
			continue
		}

		if !info.IsBlockchainSynced || !info.IsSynced {
			ShowFirstCheck()
			continue
		}

		break
	}

	fmt.Printf("\nFinish Sync at: %s\n", time.Now().Format(time.RFC3339))
}

var counter = 0

func ShowFirstCheck() {
	if !isFirstCheck {
		isFirstCheck = true
		fmt.Printf("Start Sync at: %s\n", time.Now().Format(time.RFC3339))
		fmt.Print(getBlockCount())
	}
	time.Sleep(5 * time.Second)
	counter++

	if counter == 6 {
		counter = 0
		fmt.Print(getBlockCount())
	}

}

func main() {
	flag.Parse()

	wg := new(sync.WaitGroup)
	wg.Add(1)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	go func() {
		cmd := exec.Command(dashDaemonLocation)
		err := cmd.Run()
		if err != nil {
			log.Printf("ERROR %v", err)
		}
		fmt.Println("\nDaemon closed")
		wg.Done()
	}()

	go func() {
		fmt.Println("------------------")

		StartLog()
		defer FinishLog()

		WaitSync()

		if *txId != "" {
			ws, err := websocket.Dial(fmt.Sprintf("ws://localhost:%d/trigger", *wsPort), "", "http://localhost/")
			if err != nil {
				log.Printf("ERROR1 %v", err)
				return
			}

			data, err := json.Marshal(&Response{
				Type:  "trigger",
				Value: *txId,
			})
			if err != nil {
				log.Printf("ERROR2 %v", err)
				return
			}

			_, err = ws.Write(data)
			if err != nil {
				log.Printf("ERROR3 %v", err)
				return
			}

			return
		}

		processWebSocket()
	}()

	<-c
	wg.Wait()

	fmt.Println("Server closed")
}
