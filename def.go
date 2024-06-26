package main

var (
	protoVersion uint16 = 19
	worldCenter  XY     = XY{X: xyCenter, Y: xyCenter}
)

const (
	diagSpeed = 0.70710678118
	walkSpeed = 16

	xyCenter = 2147483648
	xyMax    = xyCenter * 2

	FrameSpeedNS = 133333333
	chunkDiv     = 128
	searchChunks = 6
	lagThresh    = 8
)

type PMode uint8

// Directions
type DIR uint8

const (
	/* Directions */
	DIR_S DIR = iota
	DIR_SW
	DIR_W
	DIR_NW
	DIR_N
	DIR_NE
	DIR_E
	DIR_SE
	DIR_NONE
)

const (
	// Directions
	PMODE_PASSIVE PMode = iota
	PMODE_ATTACK
	PMODE_HEAL
)

type EFF uint8

const (
	EFFECT_NONE EFF = 1 << iota
	EFFECT_HEAL
	EFFECT_HEALER
	EFFECT_ATTACK
	EFFECT_INJURED
)

type CRE uint8

const (
	CRE_IDLE CRE = iota
	CRE_ATTACK
	CRE_FLEE
	CRE_SLEEP
)

// Network commands
type CMD uint8

const (
	CMD_Init CMD = iota
	CMD_Login
	CMD_Play
	CMD_Move
	CMD_WorldUpdate
	CMD_Chat
	CMD_Command
	CMD_PlayerMode

	CMD_WorldData
	CMD_PlayerNamesComp
	CMD_EditPlaceItem
	CMD_EditDeleteItem
)

// Used for debug messages, this could be better
var cmdNames map[CMD]string

func init() {
	cmdNames = make(map[CMD]string)
	cmdNames[CMD_Init] = "CMD_Init"
	cmdNames[CMD_Login] = "CMD_Login"
	cmdNames[CMD_Play] = "CMD_Play"
	cmdNames[CMD_Move] = "CMD_Move"
	cmdNames[CMD_WorldUpdate] = "CMD_WorldUpdate"
	cmdNames[CMD_Chat] = "CMD_Chat"
	cmdNames[CMD_Command] = "CMD_Command"
	cmdNames[CMD_PlayerMode] = "CMD_PlayerMode"

	cmdNames[CMD_WorldData] = "CMD_WorldData"
	cmdNames[CMD_PlayerNamesComp] = "CMD_PlayerNamesComp"
	cmdNames[CMD_EditPlaceItem] = "CMD_EditPlaceItem"
	cmdNames[CMD_EditDeleteItem] = "CMD_EditDeleteItem"
}
