package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
)

const FrameSpeedNS = 66666666

func processGame() {
	var gameTick uint64
	go func() {

		var buf []byte
		outbuf := bytes.NewBuffer(buf)

		for {
			gameTick++
			loopStart := time.Now()

			outbuf.Reset()

			pListLock.Lock()
			var numPlayers uint32 = uint32(len(playerList))
			binary.Write(outbuf, binary.LittleEndian, &numPlayers)

			for _, player := range playerList {
				binary.Write(outbuf, binary.LittleEndian, &player.id)
				binary.Write(outbuf, binary.LittleEndian, &player.location.pos.X)
				binary.Write(outbuf, binary.LittleEndian, &player.location.pos.Y)
			}
			for _, player := range playerList {
				writeToPlayer(player, CMD_UPDATE, outbuf.Bytes())
			}
			pListLock.Unlock()

			took := time.Since(loopStart)
			remaining := (time.Nanosecond * FrameSpeedNS) - took

			outspeed := (outbuf.Len() * int(numPlayers)) * 15.0 / 1024 / 1024 * 8
			updateSize := outbuf.Len() / 1024

			if remaining > 0 { /*Kill remaining time*/
				time.Sleep(remaining)
				if gameTick%75 == 0 {
					fmt.Printf("took: %v: out: %v mbit (%vkb)\n", took, outspeed, updateSize)
				}

			} else { /*We are lagging behind realtime*/
				doLog(true, "Unable to keep up: took: %v, out: %v mbit (%vkb", took, outspeed, updateSize)
			}

		}
	}()

	//return
	for i := 0; i < 100; i++ {
		startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(1280)), Y: uint32(int(xyHalf) + rand.Intn(1280))}
		player := &playerData{lastPing: time.Now(), id: makePlayerID(), location: locationData{pos: startLoc}}
		pListLock.Lock()
		playerList[player.id] = player
		pListLock.Unlock()
	}
}
