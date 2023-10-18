package main

import (
	"time"

	"github.com/gorilla/websocket"
)

type playerData struct {
	conn     *websocket.Conn
	id       uint32
	lastPing time.Time
}
