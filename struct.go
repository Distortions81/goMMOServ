package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

const assetArraySize = 255

type IID struct {
	Section uint8
	Num     uint8
}

type worldObject struct {
	ID  IID
	Pos XY
}

type playerData struct {
	conn *websocket.Conn

	name    string
	health  int16
	injured bool

	id            uint32
	pos           XYf32
	dir           DIR
	lastDirUpdate uint64
	mode          PMode

	effect   EFF
	target   *playerData
	targeter *playerData

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
	chunkLock sync.RWMutex

	WorldObjects []*worldObject
	players      []*playerData

	bufferFrame uint64

	pBufCount    uint16
	playerBuffer []byte

	oBufCount uint16
	objBuffer []byte

	cleanme bool
}
