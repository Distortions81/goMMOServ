package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

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

	name   string
	health int16

	id            uint32
	pos           XYf32
	dir           DIR
	lastDirUpdate uint64
	mode          PMode

	visCache []visCacheData

	effects EFF
	targets []*targetingData

	area  *areaData
	VALID bool
}

type visCacheData struct {
	pos     XY
	lastSaw uint64
}

type targetingData struct {
	target        *playerData
	targetEffects EFF
	selfEffects   EFF
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

	wBufCount      uint16
	worldObjBuffer []byte
	worldObjDirty  bool

	pBufCount    uint16
	playerBuffer []byte
}
