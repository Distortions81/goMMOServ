package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type playerData struct {
	conn *websocket.Conn
	id   uint32
	pos  XY

	lock sync.Mutex
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
	superChunks []superChunk
}

type superChunk struct {
	chunks []chunkData
}

type chunkData struct {
	objects []*objectData
}

type objectData struct {
	name string
	uid  uint32
}
