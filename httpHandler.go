package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sasha-s/go-deadlock"
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
	numConnectionsLock deadlock.Mutex

	playerList map[uint32]*playerData
	pListLock  deadlock.RWMutex

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
	player := &playerData{conn: conn, id: makePlayerID(), pos: startLoc, area: &testArea, health: 100}
	pListLock.Lock()
	playerList[player.id] = player
	addPlayerToWorld(&testArea, startLoc, player)
	pListLock.Unlock()

	conn.SetReadLimit(int64(maxNetRead))

	addConnection()
	for {
		_, data, err := conn.ReadMessage()

		if err != nil {
			doLog(true, "Error on connection read: %v", err)
			removePlayer(player, "connection lost")
			return
		}
		newParser(data, player)
	}
}

func removePlayer(player *playerData, reason string) {
	defer reportPanic("removePlayer")

	if player == nil {
		return
	}
	playerID := player.id
	killConnection(player, true)

	pListLock.Lock()
	removePlayerWorld(player.area, player.pos, player)
	delete(playerList, player.id)
	pListLock.Unlock()

	reasonStr := fmt.Sprintf("Player-%v left the game. (%v)", playerID, reason)
	send_chat(reasonStr)
}

func killConnection(player *playerData, force bool) {
	defer reportPanic("killConnection")

	if player.conn != nil {
		err := player.conn.Close()
		if err == nil || force {
			numConnectionsLock.Lock()
			if numConnections > 0 {
				numConnections--
			}
			numConnectionsLock.Unlock()
		}
		player.conn = nil
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
