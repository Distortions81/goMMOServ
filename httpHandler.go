package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{EnableCompression: false}

func gsHandler(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	go handleConnection(c)
}

func siteHandler(w http.ResponseWriter, r *http.Request) {
	fileServer.ServeHTTP(w, r)
}

const scPrefix = "<html><head><title>goSnake Scoreboard</title>"
const colsSetup = "<style>.column{float:left}.left{width:20%}.middle{width:30%}.right{width:50%}.row:after{content:\"\";display:table;clear:both}</style>"
const header = "<script>function autoRefresh(){window.location=window.location.href;}setInterval('autoRefresh()', 5000);</script></head><body bgcolor=black>"
const cols = "<div class=\"row\"><div class=\"column left\" style=\"color:white;font-size:150%%;\"><h2>%v</h2><p>%v</p></div><div class=\"column middle\" style=\"color:white;font-size:150%%;\"><h2>%v</h2><p>%v</p></div><div class=\"column right\" style=\"color:white;font-size:150%%;\"><h2>%v</h2><p>%v</p></div></div>"
const scSuffix = "</p></body></html>"

var numConnections int = 0
var numConnectionsLock sync.Mutex

func handleConnection(conn *websocket.Conn) {
	if conn == nil {
		return
	}

	addConnection()
	for {
		_, _, err := conn.ReadMessage()

		if err != nil {
			doLog(true, "Error on connection read: %v", err)

			killConnection(conn, true)
			break
		}
		//newParser(data, player)
	}
}

func killConnection(conn *websocket.Conn, force bool) {
	if conn != nil {
		err := conn.Close()
		if err == nil || force {
			numConnectionsLock.Lock()
			if numConnections > 0 {
				numConnections--
			}
			numConnectionsLock.Unlock()
		}
		conn = nil
	}
}

func getNumberConnections() int {
	numConnectionsLock.Lock()
	defer numConnectionsLock.Unlock()

	return numConnections
}

func addConnection() {
	numConnectionsLock.Lock()
	numConnections++
	numConnectionsLock.Unlock()
}
