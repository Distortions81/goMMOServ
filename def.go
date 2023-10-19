package main

/* Directions */
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

/* Network commands */
type CMD uint8

const (
	CMD_INIT CMD = iota
	CMD_LOGIN
	CMD_PLAY
	CMD_MOVE
)

/* Used for debug messages, this could be better */
var cmdNames map[CMD]string

func init() {
	cmdNames = make(map[CMD]string)
	cmdNames[CMD_INIT] = "CMD_INIT"
	cmdNames[CMD_LOGIN] = "CMD_LOGIN"
	cmdNames[CMD_PLAY] = "CMD_PLAY"
	cmdNames[CMD_MOVE] = "CMD_MOVE"
}

const xyHalf = 2147483648

var xyCenter XY = XY{X: xyHalf, Y: xyHalf}
