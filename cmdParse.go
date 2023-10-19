package main

import "github.com/gorilla/websocket"

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
		cmd_init(player, &data)

	default:
		doLog(true, "Received invalid command: 0x%02X, %v", d, string(data))
		killConnection(player.conn, false)

		player.conn = nil
		delete(playerList, player.id)
		return
	}
}

func cmd_init(player *playerData, data *[]byte) {
	defer reportPanic("cmd_init")

	writeToPlayer(player, CMD_LOGIN, nil)
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
