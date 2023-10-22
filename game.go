package main

import (
	"bytes"
	"encoding/binary"
	"time"
)

const FrameSpeedNS = 66666666

func processGame() {
	go func() {
		for {
			loopStart := time.Now()

			var buf []byte
			outbuf := bytes.NewBuffer(buf)

			pListLock.Lock()
			var numPlayers uint32 = uint32(len(playerList))
			binary.Write(outbuf, binary.LittleEndian, &numPlayers)

			for _, player := range playerList {
				player.lock.Lock()
				binary.Write(outbuf, binary.LittleEndian, &player.id)
				binary.Write(outbuf, binary.LittleEndian, &player.location.pos.X)
				binary.Write(outbuf, binary.LittleEndian, &player.location.pos.Y)
				player.lock.Unlock()
			}
			for _, player := range playerList {
				writeToPlayer(player, CMD_UPDATE, outbuf.Bytes())
			}
			pListLock.Unlock()

			took := time.Since(loopStart)
			remaining := (time.Nanosecond * FrameSpeedNS) - took

			if remaining > 0 { /*Kill remaining time*/
				time.Sleep(remaining)

			} else { /*We are lagging behind realtime*/
				time.Sleep(time.Millisecond)
				doLog(true, "Unable to keep up: took: %v", took)
			}

		}
	}()
}
