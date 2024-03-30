package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type IID struct {
	Section uint8
	Num     uint8
	Sprite  uint8
	UID     uint32
}

type worldObject struct {
	ID  IID
	Pos XY
}

type playerData struct {
	conn         *websocket.Conn
	creatureData *creatureData

	name   string
	health int16

	id            uint32
	pos           XYf32
	moveDir       DIR
	dir           DIR
	lastDirUpdate uint64
	mode          PMode

	visCache map[XY]*visCacheData
	numVis   int

	effects    EFF
	targets    []*targetingData
	numTargets int

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

type creatureData struct {
	id     IID
	mode   CRE
	target *playerData
}

type chunkData struct {
	numWorldObjects uint8
	WorldObjects    []*worldObject
	numPlayers      uint8
	players         []*playerData
	numCreatures    uint8
	creatrues       []*playerData

	playerCache []byte
	pCacheTick  uint64

	objectCache   []byte
	hasOcache     bool
	creatureCache []byte
	cCacheTick    uint64
}
