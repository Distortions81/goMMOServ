package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

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
	default:
		doLog(true, "Received invalid command: 0x%02X, %v", d, string(data))
		killConnection(player.conn, false)

		player.conn = nil
		pListLock.Lock()
		delete(playerList, player.id)
		pListLock.Unlock()

		return
	}
}

func cmd_init(player *playerData, data []byte) {
	defer reportPanic("cmd_init")

	var buf []byte
	outbuf := bytes.NewBuffer(buf)

	binary.Write(outbuf, binary.BigEndian, &player.id)

	writeToPlayer(player, CMD_LOGIN, outbuf.Bytes())
}

func cmd_chat(player *playerData, data []byte) {
	pListLock.Lock()
	defer pListLock.Unlock()

	for _, player := range playerList {
		writeToPlayer(player, CMD_CHAT, data)
	}
}

func cmd_move(player *playerData, data []byte) {

	inbuf := bytes.NewBuffer(data)

	newPos := XY{}
	binary.Read(inbuf, binary.BigEndian, &newPos)
	binary.Read(inbuf, binary.BigEndian, &newPos)

	player.lock.Lock()
	defer player.lock.Unlock()

	for t, target := range playerList {
		if t == player.id {
			continue
		}
		dist := distance(target.location.pos, newPos)
		if dist < 1 {
			fmt.Printf("Items inside each other! %v and %v\n", target.id, player.id)
			player.location.pos.X += uint32(dist) * 2
			player.location.pos.Y += uint32(dist) * 2
		} else if dist < 24 {
			fmt.Printf("BONK! #%v and #%v (%v p)\n", target.id, player.id, dist)
			return
		}
	}

	player.location.pos = newPos

	//doLog(true, "Moved to: %v,%v", player.location.pos.X, player.location.pos.Y)
}

func writeToPlayer(player *playerData, header CMD, input []byte) bool {

	if player == nil {
		return false
	}
	if player.conn == nil {

		return false
	}

	player.lock.Lock()
	defer player.lock.Unlock()

	var err error
	if input == nil {
		err = player.conn.WriteMessage(websocket.BinaryMessage, []byte{byte(header)})
	} else {
		err = player.conn.WriteMessage(websocket.BinaryMessage, append([]byte{byte(header)}, input...))
	}
	if err != nil {
		doLog(true, "Error writing response: %v", err)
		killConnection(player.conn, false)
		player.conn = nil

		delete(playerList, player.id)

		return false
	}
	return true
}
