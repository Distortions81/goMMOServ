package main

import (
	"flag"
	"fmt"
	"net/http"
)

var (
	fileServer http.Handler
	wasmMode   = false
)

const (
	version   = "0.0.1"
	buildInfo = "dev"
)

func main() {
	defer reportPanic("main")

	//Parse launch params
	devMode := flag.Bool("dev", false, "dev mode enable")
	bindIP := flag.String("ip", "", "IP to bind to")
	bindPort := flag.Int("port", 443, "port to bind to for HTTPS"
	flag.Parse()

	//Start logger
	startLog()
	logDaemon()

	/* Download server start */
	fileServer = http.FileServer(http.Dir("www"))

	/* Create HTTPS server */
	server := &http.Server{}
	server.Addr = fmt.Sprintf("%v:%v", *bindIP, *bindPort)

	/* HTTPS server */
	http.HandleFunc("/gs", gsHandler) //websocket
	http.HandleFunc("/", siteHandler) //wasm download site

	//If not in development mode, setup an origin check
	if !*devMode {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if !*devMode && origin != "https://gommo.go-game.net" {
				doLog(true, "Connection failed origin check: %v", r.RemoteAddr)
				return false
			}
			return true
		}
	}

	/* Start server*/
	doLog(true, "Starting server...")
	err := server.ListenAndServeTLS("fullchain.pem", "privkey.pem")
	if err != nil {
		doLog(true, "ListenAndServeTLS: %v", err)
		return
	}

	doLog(true, "Goodbye.")
}
