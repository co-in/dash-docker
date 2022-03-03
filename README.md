DON`T USE IN PRODUCTION. ONLY FOR EXAMPLE EASY INSTANT SEND INTEGRATION.

1. Install dependencies ```apt install openssl make wget unzip docker.io```
2. Run ```make```
3. Wait Sync completed (This is the worst moment. But that's blockchain works)
4. Visit http://127.0.0.1:8080

Blockchain & wallet stored on mounted volume ./blockchain Website stored on mounted volume ./src/frontend

How it works?

1. Init (create blockchain volume, generate dash.conf, download bootstrap.dat)
2. Frontend Container (stop prev container, pull apache image, mount frontend volume, run in background)
3. Backend Container (stop prev container, build from Dockerfile, mount blockchain volume, run)

Donate: Xgzpd6doencq7ihrz9nvDrYEUaRTfDKm2Q
