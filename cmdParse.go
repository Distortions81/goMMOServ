package main

func newParser(input []byte, player *playerData) {

	inputLen := len(input)

	if inputLen <= 0 {
		return
	}

	d := CMD(input[0])
	data := input[1:]

	cmdName := cmdNames[d]
	if cmdName != "" && d != CMD_PINGPONG {
		doLog(true, "ID: %v, Received: %v", player.id, cmdName)
	}
	switch d {
	case CMD_INIT: /*INIT*/
		cmd_init(player, &data)
	case CMD_PINGPONG: /*PING*/
		cmd_pingpong(player, &data)
	default:
		doLog(true, "Received invalid command: 0x%02X, %v", d, string(data))
		killConnection(player.conn, false)

		player.conn = nil
		delete(playerList, player.id)
		return
	}
}

func cmd_init(player *playerData, data *[]byte) {

}

func cmd_pingpong(player *playerData, data *[]byte) {

}
