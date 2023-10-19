package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const xyHalf = 2147483648

var xyCenter XY = XY{X: xyHalf, Y: xyHalf}

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
