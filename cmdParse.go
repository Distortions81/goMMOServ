package main

import (
	"bytes"
	"encoding/binary"

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
	default:
		doLog(true, "Received invalid command: 0x%02X, %v", d, string(data))
		killConnection(player.conn, false)

		player.conn = nil
		delete(playerList, player.id)
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

func cmd_move(player *playerData, data []byte) {

	inbuf := bytes.NewBuffer(data)

	binary.Read(inbuf, binary.BigEndian, &player.location.pos.X)
	binary.Read(inbuf, binary.BigEndian, &player.location.pos.Y)

	doLog(true, "Moved to: %v,%v", player.location.pos.X, player.location.pos.Y)
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
