package main

var protoVersion uint16 = 5

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
	CMD_UPDATE
	CMD_CHAT
	CMD_COMMAND
	CMD_PLAYERNAMES
	CMD_EDITPLACEITEM
)

/* Used for debug messages, this could be better */
var cmdNames map[CMD]string

func init() {
	cmdNames = make(map[CMD]string)
	cmdNames[CMD_INIT] = "CMD_INIT"
	cmdNames[CMD_LOGIN] = "CMD_LOGIN"
	cmdNames[CMD_PLAY] = "CMD_PLAY"
	cmdNames[CMD_MOVE] = "CMD_MOVE"
	cmdNames[CMD_UPDATE] = "CMD_UPDATE"
	cmdNames[CMD_CHAT] = "CMD_CHAT"
	cmdNames[CMD_COMMAND] = "CMD_COMMAND"
	cmdNames[CMD_PLAYERNAMES] = "CMD_PLAYERNAMES"
	cmdNames[CMD_EDITPLACEITEM] = "CMD_EDITPLACEITEM"
}

const xyHalf = 2147483648
