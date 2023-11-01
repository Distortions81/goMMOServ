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
	conn *websocket.Conn

	chunkList []XY

	name   string
	health int8

	id   uint32
	pos  XY
	area *areaData
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

	Name   string
	ID     uint16
	Chunks map[XY]*chunkData
	dirty  bool

	areaLock sync.RWMutex
}

type chunkData struct {
	WorldObjects []*worldObject
	players      []*playerData
}
