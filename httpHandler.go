package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{EnableCompression: false}

func gsHandler(w http.ResponseWriter, r *http.Request) {
	defer reportPanic("gsHandler")

	c, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	go handleConnection(c)
}

func siteHandler(w http.ResponseWriter, r *http.Request) {
	defer reportPanic("siteHandler")

	fileServer.ServeHTTP(w, r)
}

var (
	numConnections     int = 0
	numConnectionsLock sync.Mutex

	playerList map[uint32]*playerData
	pListLock  sync.RWMutex

	maxNetRead     = 1024 * 100
	maxConnections = 1000
)

func handleConnection(conn *websocket.Conn) {
	defer reportPanic("handleConnection")

	if conn == nil {
		return
	}

	if getNumberConnections() > maxConnections {
		return
	}

	startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(128)), Y: uint32(int(xyHalf) + rand.Intn(128))}
	player := &playerData{conn: conn, lastPing: time.Now(), id: makePlayerID(), location: locationData{pos: startLoc}}
	pListLock.Lock()
	playerList[player.id] = player
	pListLock.Unlock()

	conn.SetReadLimit(int64(maxNetRead))

	addConnection()
	for {
		_, data, err := conn.ReadMessage()

		if err != nil {
			doLog(true, "Error on connection read: %v", err)

			killConnection(conn, true)

			player.conn = nil

			pListLock.Lock()
			delete(playerList, player.id)
			pListLock.Unlock()

			break
		}
		newParser(data, player)
	}
}

func killConnection(conn *websocket.Conn, force bool) {
	defer reportPanic("killConnection")

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
	defer reportPanic("getNumberConnections")

	numConnectionsLock.Lock()
	defer numConnectionsLock.Unlock()

	return numConnections
}

func addConnection() {
	defer reportPanic("addConnection")

	numConnectionsLock.Lock()
	numConnections++
	numConnectionsLock.Unlock()
}
