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
	case CMD_SCREENSIZE: /* Vision */
		cmd_screensize(player, data)
	case CMD_COMMAND:
		cmd_command(player, data)
	default:
		doLog(true, "Received invalid command: 0x%02X, %v", d, string(data))
		removePlayer(player, "INVALID COMMAND")

		return
	}
}

func sendPlayernames(player *playerData, setName bool) {
	defer reportPanic("sendPlayernames")

	var buf []byte
	outbuf := bytes.NewBuffer(buf)

	var numNames uint32
	for _, player := range playerList {
		if player.name == "" {
			continue
		}
		numNames++
	}
	if numNames == 0 {
		return
	}

	binary.Write(outbuf, binary.LittleEndian, &numNames)

	for _, target := range playerList {
		if target.name == "" {
			continue
		}
		if setName {
			if target.id != player.id {
				continue
			}
		}
		binary.Write(outbuf, binary.LittleEndian, &target.id)

		var nameLen uint16 = uint16(len(target.name))
		binary.Write(outbuf, binary.LittleEndian, &nameLen)

		for x := 0; x < int(nameLen); x++ {
			var playerRune = rune(target.name[x])
			binary.Write(outbuf, binary.LittleEndian, &playerRune)
		}
	}

	if setName {
		for _, target := range playerList {
			writeToPlayer(target, CMD_PLAYERNAMES, outbuf.Bytes())
		}
	} else {
		writeToPlayer(player, CMD_PLAYERNAMES, outbuf.Bytes())
	}
}

func cmd_command(player *playerData, data []byte) {
	defer reportPanic("cmd_command")

	str := string(data)

	if !strings.HasPrefix(str, "/") {
		writeToPlayer(player, CMD_COMMAND, []byte("Commmands must begin with: /  (try /help)"))
		return
	}
	words := strings.Split(str, " ")
	numWords := len(words)

	if numWords < 2 {
		writeToPlayer(player, CMD_COMMAND, []byte("Commands: /name playerName"))
		return
	}

	allParams := strings.Join(words[1:], " ")
	allParamLen := len(allParams)

	command := strings.TrimPrefix(words[0], "/")

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

func cmd_screensize(player *playerData, data []byte) {
	defer reportPanic("cmd_screensize")

}

func cmd_init(player *playerData, data []byte) {
	defer reportPanic("cmd_init")

	inbuf := bytes.NewBuffer(data)
	var version uint16
	binary.Read(inbuf, binary.LittleEndian, &version)
	if version != protoVersion {
		doLog(true, "Invalid proto version: %v", version)
		writeToPlayer(player, CMD_INIT, []byte{})
		return
	}

	var buf []byte
	outbuf := bytes.NewBuffer(buf)

	binary.Write(outbuf, binary.LittleEndian, &player.id)

	writeToPlayer(player, CMD_LOGIN, outbuf.Bytes())

	cmd_move(player, []byte{})

	welcomeStr := fmt.Sprintf("Player-%v joined the game.", player.id)
	send_chat(welcomeStr)
}

const maxChat = 256

func cmd_chat(player *playerData, data []byte) {
	defer reportPanic("cmd_chat")
	if len(data) > maxChat {
		return
	}

	pListLock.RLock()
	defer pListLock.RUnlock()

	pName := fmt.Sprintf("Player-%v says: %v", player.id, string(data))

	for _, target := range playerList {
		if target.conn == nil {
			continue
		}
		writeToPlayer(target, CMD_CHAT, []byte(pName))
	}
}

func send_chat(data string) {
	defer reportPanic("send_chat")
	pListLock.RLock()
	defer pListLock.RUnlock()

	for _, target := range playerList {
		if target.conn == nil {
			continue
		}
		writeToPlayer(target, CMD_CHAT, []byte(data))
	}
}

func cmd_move(player *playerData, data []byte) {
	defer reportPanic("cmd_move")
	pListLock.Lock()
	defer pListLock.Unlock()

	inbuf := bytes.NewBuffer(data)

	var newPosX, newPosY int8
	binary.Read(inbuf, binary.LittleEndian, &newPosX)
	binary.Read(inbuf, binary.LittleEndian, &newPosY)

	var newPos XY = XY{X: uint32(int(player.pos.X) + int(newPosX)),
		Y: uint32(int(player.pos.Y) + int(newPosY))}

	for x := -2; x < 2; x++ {
		for y := -2; y < 2; y++ {
			chunkPos := XY{X: uint32(int(player.pos.X/chunkDiv) + x),
				Y: uint32(int(player.pos.Y/chunkDiv) + y)}
			chunk := player.area.chunks[chunkPos]
			if chunk == nil {
				continue
			}

			for _, target := range chunk.players {
				if target.id == player.id {
					//Skip self
					continue
				}
				dist := distance(target.pos, newPos)

				if dist < 10 {
					fmt.Printf("Items inside each other! %v and %v (%v p)\n", target.id, player.id, dist)
					newPos.X += 24
					newPos.Y += 24
					movePlayer(player.area, newPos, player)

					return
				} else if dist < 24 {
					fmt.Printf("BONK! #%v and #%v (%v p)\n", target.id, player.id, dist)
					return
				}

			}
		}
	}

	movePlayer(player.area, newPos, player)

	//doLog(true, "Move: %v,%v", newPosX, newPosY)
}

func writeToPlayer(player *playerData, header CMD, input []byte) bool {
	//defer reportPanic("writeToPlayer")
	if player == nil {
		return false
	}

	player.connLock.Lock()
	defer player.connLock.Unlock()

	if player.conn == nil {
		return false
	}

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
