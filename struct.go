package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type playerData struct {
	conn *websocket.Conn
	id   uint32

	name     string
	location locationData

	inventory []objectData

	lastPing time.Time
	lock     sync.Mutex
}

type locationData struct {
	pos XY

	areaid uint32

	areaP       *areaData
	superChunkP *superChunk
	chunkP      *chunkData
}

type XY struct {
	X uint32
	Y uint32
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
