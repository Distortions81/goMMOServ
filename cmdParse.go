package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"

	"github.com/gorilla/websocket"
)

func newParser(input []byte, player *playerData) {
	defer reportPanic("newParser")

	processLock.Lock()
	defer processLock.Unlock()

	inputLen := len(input)

	if inputLen <= 0 {
		return
	}

	d := CMD(input[0])
	data := input[1:]

	if d != CMD_Move {
		cmdName := cmdNames[d]
		doLog(true, "ID: %v, Received: %v, Data: %v", player.id, cmdName, string(data))
	}

	switch d {
	case CMD_Init:
		cmd_init(player, data)
		sendPlayernames(player, false)
	case CMD_Move:
		cmd_move(player, data)
	case CMD_Chat:
		cmd_chat(player, data)
	case CMD_Command:
		cmd_command(player, data)
	case CMD_PlayerMode:
		cmd_playermode(player, data)
	case CMD_EditPlaceItem:
		cmd_editPlaceItem(player, data)
	case CMD_EditDeleteItem:
		cmd_editDeleteItem(player, data)
	default:
		doLog(true, "Received invalid command: 0x%02X, %vb", d, len(data))
		removePlayer(player, "INVALID COMMAND")

		return
	}
}

func cmd_playermode(player *playerData, data []byte) {
	defer reportPanic("cmd_playermode")

	inbuf := bytes.NewBuffer(data)
	for _, t := range player.targets {
		removeTarget(player, t.target)
	}

	binary.Read(inbuf, binary.LittleEndian, &player.mode)
}

func cmd_editDeleteItem(player *playerData, data []byte) {
	defer reportPanic("cmd_editPlaceItem")

	inbuf := bytes.NewBuffer(data)

	var sectionID, itemID, spriteID uint8
	var editPosX, editPosY uint32

	binary.Read(inbuf, binary.LittleEndian, &sectionID)
	binary.Read(inbuf, binary.LittleEndian, &itemID)
	binary.Read(inbuf, binary.LittleEndian, &spriteID)
	binary.Read(inbuf, binary.LittleEndian, &editPosX)
	binary.Read(inbuf, binary.LittleEndian, &editPosY)

	pos := XY{X: editPosX, Y: editPosY}

	doLog(true, "%v:%v:%v %v,%v", sectionID, itemID, spriteID, editPosX, editPosY)

	removeWorldObject(areaList[0], pos, IID{Section: sectionID, Num: itemID, Sprite: spriteID})
	areaList[0].dirty = true
}

func cmd_editPlaceItem(player *playerData, data []byte) {
	defer reportPanic("cmd_editPlaceItem")

	if player == nil || player.area == nil {
		return
	}

	inbuf := bytes.NewBuffer(data)

	var editSectionID, editItemID, editSpriteID uint8
	var editPosX, editPosY uint32

	binary.Read(inbuf, binary.LittleEndian, &editSectionID)
	binary.Read(inbuf, binary.LittleEndian, &editItemID)
	binary.Read(inbuf, binary.LittleEndian, &editSpriteID)
	binary.Read(inbuf, binary.LittleEndian, &editPosX)
	binary.Read(inbuf, binary.LittleEndian, &editPosY)

	pos := XY{X: editPosX, Y: editPosY}
	newObj := &worldObject{ID: IID{Section: editSectionID, Num: editItemID, Sprite: editSpriteID}, Pos: pos}
	doLog(true, "%v:%v:%v %v,%v", editSectionID, editItemID, editSectionID, editPosX, editPosY)
	addWorldObject(player.area, pos, newObj)
	player.area.dirty = true
}

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

		compBuf := CompressZip(outbuf.Bytes())
		for _, target := range playerList {
			writeToPlayer(target, CMD_PlayerNamesComp, compBuf)
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

		writeToPlayer(player, CMD_PlayerNamesComp, CompressZip(outbuf.Bytes()))
	}
}

func cmd_command(player *playerData, data []byte) {
	defer reportPanic("CMD_Command")

	//Make a string out of the data
	str := string(data)

	//Check if command has prefix
	if !strings.HasPrefix(str, "/") {
		writeToPlayer(player, CMD_Command, []byte("Commmands must begin with: /  (try /help)"))
		return
	}

	//Split into args
	words := strings.Split(str, " ")
	numWords := len(words)

	//Check if enough args
	if numWords < 2 {
		writeToPlayer(player, CMD_Command, []byte("Commands:"))
		writeToPlayer(player, CMD_Command, []byte("/name NewName"))
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
			writeToPlayer(player, CMD_Command, []byte("Name not long enough."))
			return
		} else if allParamLen > 32 {
			writeToPlayer(player, CMD_Command, []byte("Name too long."))
			return
		}
		player.name = allParams
		writeToPlayer(player, CMD_Command, []byte("Name set."))
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
		writeToPlayer(player, CMD_Init, []byte{})
		removePlayer(player, "invalid version")
		return
	}
	addPlayerToWorld(player.area, player.pos, player)

	for !movePlayer(player, false) {
		tryPos := XYf32{X: float32(halfArea - rand.Intn(spawnArea)),
			Y: float32(halfArea - rand.Intn(spawnArea))}
		movePlayerChunk(player.area, tryPos, player)
		fmt.Println("Spawn blocked... Trying again.")
	}

	var buf []byte
	outbuf := bytes.NewBuffer(buf)

	//Send player id
	binary.Write(outbuf, binary.LittleEndian, &player.id)
	binary.Write(outbuf, binary.LittleEndian, &player.area.ID)
	writeToPlayer(player, CMD_Login, outbuf.Bytes())

	//Notify players we joined
	welcomeStr := fmt.Sprintf("%v joined the game.", player.name)
	send_chat(welcomeStr)

}

const maxChat = 256

func cmd_chat(player *playerData, data []byte) {
	defer reportPanic("cmd_chat")
	if len(data) > maxChat {
		return
	}

	pName := fmt.Sprintf("%v says: %v", player.name, string(data))

	for _, target := range playerList {
		if target.conn == nil {
			continue
		}
		writeToPlayer(target, CMD_Chat, []byte(pName))
	}
}

func send_chat(data string) {
	defer reportPanic("send_chat")

	for _, target := range playerList {
		if target.conn == nil {
			continue
		}
		writeToPlayer(target, CMD_Chat, []byte(data))
	}
}

func cmd_move(player *playerData, data []byte) {
	defer reportPanic("cmd_move")

	if player.health < 1 {
		return
	}

	inbuf := bytes.NewBuffer(data)

	//Read direction
	binary.Read(inbuf, binary.LittleEndian, &player.moveDir)
	if player.moveDir != DIR_NONE {
		player.dir = player.moveDir
	}
	player.lastDirUpdate = gameTick
}

func writeToPlayer(player *playerData, header CMD, input []byte) bool {
	//defer reportPanic("writeToPlayer") (EOF causes panic)

	//Sanity check
	if player == nil || player.conn == nil {
		return false
	}

	//Log event if not update
	if header != CMD_WorldUpdate {
		cmdName := cmdNames[header]
		doLog(true, "ID: %v, Sent: %v, Data: %vb", player.id, cmdName, len(input))
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
