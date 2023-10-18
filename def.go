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
	CMD_PINGPONG

	RECV_KEYFRAME
)

/* Used for debug messages, this could be better */
var cmdNames map[CMD]string

func init() {
	cmdNames = make(map[CMD]string)
	cmdNames[CMD_INIT] = "CMD_INIT"
	cmdNames[CMD_PINGPONG] = "CMD_PINGPONG"
	cmdNames[RECV_KEYFRAME] = "RECV_KEYFRAME"
}
