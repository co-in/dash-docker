DASH_NOTIFY_PORT=8090
DASH_IMAGE=dash-coin
DASH_CONTAINER=dashpay-instant
DASH_DATA_DIR=blockchain
DASH_DATA_DIR_NET=/testnet3
DASH_BOOTSTRAP_FILE=${PWD}/${DASH_DATA_DIR}${DASH_DATA_DIR_NET}/bootstrap.dat
DASH_BOOTSTRAP_FILE_ARCHIVE=${DASH_BOOTSTRAP_FILE}.zip

FRONTEND_SERVER_PORT=8080
FRONTEND_SERVER_IMAGE=httpd
FRONTEND_SERVER_CONTAINER=apache-server

testnet:
	@echo "> START PREPARE: $$(date --rfc-3339=sec)"
	@echo "> NETWORK: TEST"
ifeq ($(wildcard ${PWD}/${DASH_DATA_DIR}/),)
	@echo "> CREATE VOLUME '${PWD}/${DASH_DATA_DIR}'"
	@mkdir -p ${PWD}/${DASH_DATA_DIR}${DASH_DATA_DIR_NET}
endif
ifeq ($(wildcard ./${DASH_DATA_DIR}/dash.conf),)
	@echo "> CREATE CONFIG '${PWD}/${DASH_DATA_DIR}/dash.conf'"
	@echo "keypool=1\n\
	#daemon=1\n\
	addressindex=1\n\
	txindex=1\n\
	#checklevel=1\n\
		#0: only check whether the block exists on disk\n\
		#1: verify block validity (default)\n\
		#2: verify transaction index validity\n\
		#3: check transaction hashes\n\
		#4: check whether spent txouts were spent within the main chain\n\
		#5: check whether all prevouts are marked spent\n\
		#6: check whether spent txouts were spent by a valid transaction that consumes them\n\
	#server=1\n\
	listen=1\n\
	usehd=1\n\
	testnet=1\n\
	instantsendnotify=${DASH_CONTAINER} --tx=%s\n\
	rpcallowip=127.0.0.1\n\
	rpcuser=$$(openssl rand -hex 32)\n\
	rpcpassword=$$(openssl rand -hex 32)" > ${PWD}/${DASH_DATA_DIR}/dash.conf
endif
ifeq ($(wildcard ${PWD}/${DASH_DATA_DIR}${DASH_DATA_DIR_NET}/blocks/index),)
	@if [ ! -f ${DASH_BOOTSTRAP_FILE} ]; then \
		echo "Bootsrap not found"; \
		if [ ! -f ${DASH_BOOTSTRAP_FILE_ARCHIVE} ]; then \
			echo "Bootsrap achive not found"; \
			for number in 0 -1 -2 -3 -4 -5 -6; do \
				DI=$$(date --date="$${dataset_date} -$${date_diff} $$number day" --rfc-3339=date); \
				echo "> TRY TO DOWNLOAD BOOTSTRAP BY DATE $$DI"; \
				wget https://dash-bootstrap.ams3.digitaloceanspaces.com/testnet/$$DI/bootstrap.dat.zip -q --show-progress  -O ${DASH_BOOTSTRAP_FILE_ARCHIVE} || continue; \
				break; \
			done; \
		fi; \
		if [ -f ${DASH_BOOTSTRAP_FILE_ARCHIVE} ]; then \
			echo "> EXTRACT BOOTSTRAP ARCHIVE"; \
			unzip ${DASH_BOOTSTRAP_FILE_ARCHIVE} -d ${PWD}/${DASH_DATA_DIR}${DASH_DATA_DIR_NET} 2>/dev/null || true; \
			echo "> REMOVE ARCHIVE"; \
			rm -rf ${DASH_BOOTSTRAP_FILE_ARCHIVE}; \
		fi; \
	fi
endif
	@echo "> STOP PREV '${FRONTEND_SERVER_CONTAINER}' FRONTEND CONTAINER"
	@docker stop ${FRONTEND_SERVER_CONTAINER} 2>/dev/null || true

	#@echo "> PULL '${FRONTEND_SERVER_IMAGE}' FRONTEND IMAGE"
	#@docker pull ${FRONTEND_SERVER_IMAGE}

	@echo "> START '${FRONTEND_SERVER_CONTAINER}' FRONTEND CONTAINER FROM IMAGE '${FRONTEND_SERVER_IMAGE}' WITH HTTP PORT :${FRONTEND_SERVER_PORT}"
	@docker run \
		--rm \
		--name ${FRONTEND_SERVER_CONTAINER} \
		-p ${FRONTEND_SERVER_PORT}:80 \
		-v "${PWD}/src/frontend":/usr/local/apache2/htdocs/ \
		-d ${FRONTEND_SERVER_IMAGE}

	@echo "> GRACEFULL STOP PREV '${DASH_CONTAINER}' BACKEND CONTAINER"
	@docker kill ----signal=SIGINT ${DASH_CONTAINER} 2>/dev/null || true

	@echo "> BUILD BACKEND IMAGE '${DASH_IMAGE}'"
	@docker build \
		-t ${DASH_IMAGE}:latest \
		--build-arg BINARY=${DASH_CONTAINER} \
		./src/backend

	@echo "> FINISH PREPARE: $$(date --rfc-3339=sec)"
	@echo "> START '${DASH_CONTAINER}' BACKEND CONTAINER FROM IMAGE '${DASH_IMAGE}' WITH WS PORT :${DASH_NOTIFY_PORT}"
	@docker run \
		-it \
		--rm \
		--name ${DASH_CONTAINER} \
		-p ${DASH_NOTIFY_PORT}:8080 \
		-v ${PWD}/${DASH_DATA_DIR}:/dashd/.dashcore \
		--entrypoint ${DASH_CONTAINER} \
		${DASH_IMAGE}

