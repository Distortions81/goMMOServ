package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type worldObject struct {
	ItemId uint32
	Pos    XY
	uid    uint32
}

type playerData struct {
	conn     *websocket.Conn
	connLock sync.Mutex

	name   string
	health int8

	id    uint32
	pos   XY
	area  *areaData
	plock sync.RWMutex
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
	Version uint16

	Name     string
	ID       uint16
	arealock sync.RWMutex
	Chunks   map[XY]*chunkData
	dirty    bool
}

type chunkData struct {
	WorldObjects []*worldObject
	players      []*playerData
	chunkLock    sync.RWMutex
}
