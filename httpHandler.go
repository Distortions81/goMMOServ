package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"

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
	numConnections atomic.Int32
	playerList     []*playerData
	playerListLock sync.Mutex

	maxNetRead           = 1024 * 1000
	maxConnections int32 = 1000
)

func redirectToTls(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://gommo.go-game.net/"+r.RequestURI, http.StatusMovedPermanently)
}

func handleConnection(conn *websocket.Conn) {
	defer reportPanic("handleConnection")

	if conn == nil {
		return
	}

	if numConnections.Load() > maxConnections {
		return
	}

	startLoc := XYf32{X: float32(rand.Intn(128)), Y: float32(rand.Intn(128))}
	player := &playerData{conn: conn, id: makePlayerID(), pos: startLoc, area: areaList[0], health: 100, dir: DIR_NONE}
	playerList = append(playerList, player)
	addPlayerToWorld(player.area, player.pos, player)

	conn.SetReadLimit(int64(maxNetRead))

	numConnections.Add(1)
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

	var reasonStr string
	if player.name == "" {
		reasonStr = fmt.Sprintf("Player-%v left the game. (%v)", player.id, reason)
	} else {
		reasonStr = fmt.Sprintf("%v left the game. (%v)", player.name, reason)
	}

	killConnection(player, true)
	removePlayerWorld(player.area, player.pos, player)
	deletePlayer(player)

	send_chat(reasonStr)
}

func deletePlayer(player *playerData) {

	playerListLock.Lock()
	defer playerListLock.Unlock()

	//Does not preserve order
	playerListLen := len(playerList) - 1
	for t, target := range playerList {
		if target.id == player.id {
			playerList[t] = playerList[playerListLen]
			playerList = playerList[:playerListLen]
		}
	}
}

func killConnection(player *playerData, force bool) {
	defer reportPanic("killConnection")

	if player.conn != nil {
		err := player.conn.Close()
		if err == nil || force {
			if numConnections.Load() > 0 {
				numConnections.Add(-1)
			}
		}
		player.conn = nil
	}
}
