package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	fileServer http.Handler
	areaList   map[uint16]*areaData
	gTestMode  bool
)

func main() {
	defer reportPanic("main")
	defer time.Sleep(time.Second)

	//Parse launch params
	devMode := flag.Bool("dev", false, "dev mode enable")
	bindIP := flag.String("ip", "", "IP to bind to")
	bindPort := flag.Int("port", 443, "port to bind to for HTTPS")
	testMode := flag.Bool("test", false, "load many test characters")
	flag.Parse()

	gTestMode = *testMode

	//Start logger
	startLog()
	logDaemon()

	/* make test area */
	tmp := &areaData{Name: "test", ID: 0, Chunks: make(map[XY]*chunkData)}
	areaList = make(map[uint16]*areaData)
	areaList[tmp.ID] = tmp
	loadWorld()

	playerList = make(map[uint32]*playerData)

	processGame()

	/* Download server start */
	fileServer = http.FileServer(http.Dir("www"))

	/* Create HTTPS server */
	server := &http.Server{}
	server.Addr = fmt.Sprintf("%v:%v", *bindIP, *bindPort)

	/* HTTPS server */
	http.HandleFunc("/gs", gsHandler) //websocket
	http.HandleFunc("/", siteHandler) //wasm download site

	//If not in development mode, setup an origin check
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if !*devMode && origin != "https://gommo.go-game.net" {
			doLog(true, "Connection failed origin check: %v", r.RemoteAddr)
			return false
		}
		return true
	}

	/* Start server*/
	doLog(true, "Starting server...")

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("%v:80", *bindIP), http.HandlerFunc(redirectToTls)); err != nil {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	err := server.ListenAndServeTLS("fullchain.pem", "privkey.pem")
	if err != nil {
		doLog(true, "ListenAndServeTLS: %v", err)
		return
	}

	doLog(true, "Goodbye.")
}
