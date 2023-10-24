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

	cmdName := cmdNames[d]
	doLog(true, "ID: %v, Received: %v", player.id, cmdName)

	switch d {
	case CMD_INIT: /*INIT*/
		cmd_init(player, data)
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

func cmd_command(player *playerData, data []byte) {
	str := string(data)

	if !strings.HasPrefix(str, "/") {
		writeToPlayer(player, CMD_COMMAND, []byte("Commmands must begin with /"))
		return
	}
	words := strings.Split(str, " ")
	numWords := len(words)

	if numWords < 2 {
		return
	}

	allParams := strings.Join(words[1:], " ")
	allParamLen := len(allParams)

	command := strings.TrimPrefix(words[0], "/")

	if strings.EqualFold(command, "name") {
		if allParamLen < 3 || allParamLen > 32 {
			return
		}
		player.name = allParams
	}

}

func cmd_screensize(player *playerData, data []byte) {
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

	if player == nil {
		return false
	}
	//Not normally needed
	//player.connLock.Lock()
	//defer player.connLock.Unlock()

	if player.conn == nil {
		return false
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
