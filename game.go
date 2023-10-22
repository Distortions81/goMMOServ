package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
)

const FrameSpeedNS = 66666666
const cropSize = 512 + 24

func processGame() {
	var gameTick uint64
	go func() {

		var buf []byte
		outbuf := bytes.NewBuffer(buf)

		for {
			gameTick++
			loopStart := time.Now()

			pListLock.RLock()

			for _, player := range playerList {
				if player.conn == nil {
					continue
				}
				outbuf.Reset()
				var numPlayers uint32 = 0
				var tmpList []*playerData
				for _, target := range playerList {
					xdiff := int(player.pos.X) - int(target.pos.X)
					ydiff := int(player.pos.Y) - int(target.pos.Y)
					if xdiff > cropSize || ydiff > cropSize ||
						xdiff < -cropSize || ydiff < -cropSize {
						continue
					}
					tmpList = append(tmpList, target)
					numPlayers++
				}
				binary.Write(outbuf, binary.LittleEndian, &numPlayers)
				for _, target := range tmpList {
					binary.Write(outbuf, binary.LittleEndian, &target.id)
					binary.Write(outbuf, binary.LittleEndian, &target.pos.X)
					binary.Write(outbuf, binary.LittleEndian, &target.pos.Y)
				}
				writeToPlayer(player, CMD_UPDATE, outbuf.Bytes())
			}
			pListLock.RUnlock()

			took := time.Since(loopStart)
			remaining := (time.Nanosecond * FrameSpeedNS) - took

			if remaining > 0 { /*Kill remaining time*/
				time.Sleep(remaining)

				if gTestMode {
					if gameTick%75 == 0 {
						fmt.Printf("took: %v\n", took)
					}
				}

			} else { /*We are lagging behind realtime*/
				doLog(true, "Unable to keep up: took: %v\n", took)
			}

		}
	}()

	if gTestMode {
		for i := 0; i < 5000; i++ {
			startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(10240)), Y: uint32(int(xyHalf) + rand.Intn(10240))}
			player := &playerData{id: makePlayerID(), pos: startLoc}
			pListLock.Lock()
			playerList[player.id] = player
			pListLock.Unlock()
		}
	}
}
