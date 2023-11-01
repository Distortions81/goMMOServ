package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

func newParser(input []byte, player *playerData) {
	defer reportPanic("newParser")

	inputLen := len(input)

	if inputLen <= 0 {
		return
	}

	d := CMD(input[0])
	data := input[1:]

	if d != CMD_MOVE {
		cmdName := cmdNames[d]
		doLog(true, "ID: %v, Received: %v, Data: %v", player.id, cmdName, string(data))
	}

	switch d {
	case CMD_INIT: /*INIT*/
		cmd_init(player, data)
		sendPlayernames(player, false)
	case CMD_MOVE: /*MOVE*/
		cmd_move(player, data)
	case CMD_CHAT: /*CHAT*/
		cmd_chat(player, data)
	case CMD_COMMAND:
		cmd_command(player, data)
	case CMD_EDITPLACEITEM:
		cmd_editPlaceItem(player, data)
	case CMD_EDITDELETEITEM:
		cmd_editDeleteItem(player, data)
	default:
		doLog(true, "Received invalid command: 0x%02X, %v", d, string(data))
		removePlayer(player, "INVALID COMMAND")

		return
	}
}

func cmd_editDeleteItem(player *playerData, data []byte) {
	defer reportPanic("cmd_editPlaceItem")

	inbuf := bytes.NewBuffer(data)

	var editPosX, editPosY, editID uint32

	binary.Read(inbuf, binary.LittleEndian, &editID)
	binary.Read(inbuf, binary.LittleEndian, &editPosX)
	binary.Read(inbuf, binary.LittleEndian, &editPosY)

	pos := XY{X: editPosX, Y: editPosY}

	doLog(true, "%v: %v,%v", editID, editPosX, editPosY)
	removeWorldObject(areaList[0], pos, editID)
	areaList[0].dirty = true
}

func cmd_editPlaceItem(player *playerData, data []byte) {
	defer reportPanic("cmd_editPlaceItem")

	if player == nil || player.area == nil {
		return
	}

	inbuf := bytes.NewBuffer(data)

	var editPosX, editPosY, editID uint32

	binary.Read(inbuf, binary.LittleEndian, &editID)
	binary.Read(inbuf, binary.LittleEndian, &editPosX)
	binary.Read(inbuf, binary.LittleEndian, &editPosY)

	pos := XY{X: editPosX, Y: editPosY}
	newObj := &worldObject{uid: uint32(makeObjectID()), Pos: pos, ItemId: editID}

	doLog(true, "%v: %v,%v", editID, editPosX, editPosY)
	addWorldObject(player.area, pos, newObj)
	player.area.dirty = true
}

/* This should use a cached list */
func sendPlayernames(player *playerData, setName bool) {
	defer reportPanic("sendPlayernames")

	var buf []byte
	outbuf := bytes.NewBuffer(buf)

	var numNames uint32

	//Send to all if a player changed their name,
	//otherwise send whole list to specific player
	if setName {
		numNames = 1
		binary.Write(outbuf, binary.LittleEndian, &numNames)
		binary.Write(outbuf, binary.LittleEndian, &player.id)

		var nameLen uint16 = uint16(len(player.name))
		binary.Write(outbuf, binary.LittleEndian, &nameLen)

		for x := 0; x < int(nameLen); x++ {
			var playerRune = rune(player.name[x])
			binary.Write(outbuf, binary.LittleEndian, &playerRune)
		}

		for _, target := range playerList {
			writeToPlayer(target, CMD_PLAYERNAMES, outbuf.Bytes())
		}
	} else {

		//Count number of players that have names
		for _, player := range playerList {
			if player.name == "" {
				continue
			}
			numNames++
		}

		//Nothing to send, exit
		if numNames == 0 {
			return
		}

		//Write out header
		binary.Write(outbuf, binary.LittleEndian, &numNames)

		//Serialize player names
		for _, target := range playerList {
			if target.name == "" {
				continue
			}
			binary.Write(outbuf, binary.LittleEndian, &target.id)

			var nameLen uint16 = uint16(len(target.name))
			binary.Write(outbuf, binary.LittleEndian, &nameLen)

			for x := 0; x < int(nameLen); x++ {
				var playerRune = rune(target.name[x])
				binary.Write(outbuf, binary.LittleEndian, &playerRune)
			}
		}
		writeToPlayer(player, CMD_PLAYERNAMES, outbuf.Bytes())
	}
}

func cmd_command(player *playerData, data []byte) {
	defer reportPanic("cmd_command")

	//Make a string out of the data
	str := string(data)

	//Check if command has prefix
	if !strings.HasPrefix(str, "/") {
		writeToPlayer(player, CMD_COMMAND, []byte("Commmands must begin with: /  (try /help)"))
		return
	}

	//Split into args
	words := strings.Split(str, " ")
	numWords := len(words)

	//Check if enough args
	if numWords < 2 {
		writeToPlayer(player, CMD_COMMAND, []byte("Keybinds:"))
		writeToPlayer(player, CMD_COMMAND, []byte("[ \\ ] key -- toggle edit mode, click to place"))
		writeToPlayer(player, CMD_COMMAND, []byte("[ + ] and [ - ] keys, cycle objects in edit mode"))
		writeToPlayer(player, CMD_COMMAND, []byte("[ n ], cycle night level"))
		writeToPlayer(player, CMD_COMMAND, []byte("[ l ], toggle motion smoothing"))
		writeToPlayer(player, CMD_COMMAND, []byte("[ESC] exit Chat or Command mode."))
		writeToPlayer(player, CMD_COMMAND, []byte("[Return] or [Enter] Chat mode"))
		writeToPlayer(player, CMD_COMMAND, []byte("[ ~ ] Command mode"))

		writeToPlayer(player, CMD_COMMAND, []byte("Commands:"))
		writeToPlayer(player, CMD_COMMAND, []byte("/name NewName"))
		return
	}

	//Join args, for some command types
	allParams := strings.Join(words[1:], " ")
	allParamLen := len(allParams)

	//Remove command prefix
	command := strings.TrimPrefix(words[0], "/")

	//Commands
	if strings.EqualFold(command, "name") {
		if allParamLen < 3 {
			writeToPlayer(player, CMD_COMMAND, []byte("Name not long enough."))
			return
		} else if allParamLen > 32 {
			writeToPlayer(player, CMD_COMMAND, []byte("Name too long."))
			return
		}
		player.name = allParams
		writeToPlayer(player, CMD_COMMAND, []byte("Name set."))
		sendPlayernames(player, true)
	}
}

func cmd_init(player *playerData, data []byte) {
	defer reportPanic("cmd_init")

	inbuf := bytes.NewBuffer(data)
	var version uint16

	//Check proto version
	binary.Read(inbuf, binary.LittleEndian, &version)
	if version != protoVersion {
		doLog(true, "Invalid proto version: %v", version)
		writeToPlayer(player, CMD_INIT, []byte{})
		return
	}

	var buf []byte
	outbuf := bytes.NewBuffer(buf)

	//Send player id
	binary.Write(outbuf, binary.LittleEndian, &player.id)
	binary.Write(outbuf, binary.LittleEndian, &player.area.ID)
	writeToPlayer(player, CMD_LOGIN, outbuf.Bytes())

	addPlayerToWorld(player.area, player.pos, player)

	//Notify players we joined
	welcomeStr := fmt.Sprintf("Player-%v joined the game.", player.id)
	send_chat(welcomeStr)
}

const maxChat = 256

func cmd_chat(player *playerData, data []byte) {
	defer reportPanic("cmd_chat")
	if len(data) > maxChat {
		return
	}

	pName := fmt.Sprintf("Player-%v says: %v", player.id, string(data))
	if player.name != "" {
		pName = fmt.Sprintf("%v says: %v", player.name, string(data))
	}

	for _, target := range playerList {
		if target.conn == nil {
			continue
		}
		writeToPlayer(target, CMD_CHAT, []byte(pName))
	}
}

func send_chat(data string) {
	defer reportPanic("send_chat")

	for _, target := range playerList {
		if target.conn == nil {
			continue
		}
		writeToPlayer(target, CMD_CHAT, []byte(data))
	}
}

func cmd_move(player *playerData, data []byte) {
	defer reportPanic("cmd_move")

	inbuf := bytes.NewBuffer(data)

	//Read direction
	binary.Read(inbuf, binary.LittleEndian, &player.dir)
}

func writeToPlayer(player *playerData, header CMD, input []byte) bool {
	//defer reportPanic("writeToPlayer") (EOF causes panic)

	//Sanity check
	if player == nil || player.conn == nil {
		return false
	}

	//Log event if not update
	if header != CMD_UPDATE {
		cmdName := cmdNames[header]
		doLog(true, "ID: %v, Sent: %v, Data: %v", player.id, cmdName, string(input))
	}

	var err error
	if input == nil {
		err = player.conn.WriteMessage(websocket.BinaryMessage, []byte{byte(header)})
	} else {
		err = player.conn.WriteMessage(websocket.BinaryMessage, append([]byte{byte(header)}, input...))
	}

	if err != nil {
		doLog(true, "Error writing response: %v", err)
		removePlayer(player, "connection lost")

		return false
	}
	return true
}
