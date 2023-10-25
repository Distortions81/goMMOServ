package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type playerData struct {
	conn     *websocket.Conn
	connLock sync.Mutex

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
	chunks map[XY]*chunkData
}

type chunkData struct {
	objects []*objectData
	players []*playerData
}

type objectData struct {
	name string
	uid  uint64

	pos XY
}
