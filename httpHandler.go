package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"

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

	playerList     map[uint32]*playerData
	playerListLock sync.RWMutex

	maxNetRead     = 1024 * 1000
	maxConnections = 1000
)

func redirectToTls(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://gommo.go-game.net/"+r.RequestURI, http.StatusMovedPermanently)
}

func handleConnection(conn *websocket.Conn) {
	defer reportPanic("handleConnection")

	if conn == nil {
		return
	}

	if getNumberConnections() > maxConnections {
		return
	}

	startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(128)), Y: uint32(int(xyHalf) + rand.Intn(128))}
	player := &playerData{conn: conn, id: makePlayerID(), pos: startLoc, area: areaList[0], health: 100}
	playerListLock.Lock()
	playerList[player.id] = player
	addPlayerToWorld(player.area, startLoc, player)
	playerListLock.Unlock()

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

	playerListLock.Lock()
	removePlayerWorld(player.area, player.pos, player)
	delete(playerList, player.id)
	playerListLock.Unlock()

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
