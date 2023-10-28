package main

import (
	"github.com/gorilla/websocket"
	"github.com/sasha-s/go-deadlock"
)

type worldObject struct {
	itemId uint32
	pos    XY
	uid    uint32
}

type playerData struct {
	conn     *websocket.Conn
	connLock deadlock.Mutex

	name   string
	health int8

	id    uint32
	pos   XY
	area  *areaData
	plock deadlock.RWMutex
}

type XY struct {
	X uint32
	Y uint32
}

type XYf32 struct {
	X float32
	Y float32
}

type XYs struct {
	X int32
	Y int32
}

type areaData struct {
	id       uint16
	arealock deadlock.RWMutex
	chunks   map[XY]*chunkData
}

type chunkData struct {
	worldObjects []*worldObject
	players      []*playerData
	chunkLock    deadlock.RWMutex
}
