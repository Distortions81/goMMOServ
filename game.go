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

		var buf, cbuf []byte
		outbuf := bytes.NewBuffer(buf)
		countbuf := bytes.NewBuffer(cbuf)

		for {
			gameTick++
			loopStart := time.Now()

			pListLock.RLock()

			for _, player := range playerList {
				var numPlayers uint32

				outbuf.Reset()
				countbuf.Reset()
				for x := -numChunks; x < numChunks; x++ {
					for y := -numChunks; y < numChunks; y++ {
						chunkPos := XY{X: uint32(int(player.pos.X/chunkDiv) + x),
							Y: uint32(int(player.pos.Y/chunkDiv) + y)}
						chunk := player.area.chunks[chunkPos]
						if chunk == nil {
							continue
						}

						for _, target := range chunk.players {
							numPlayers += uint32(len(chunk.players))
							binary.Write(outbuf, binary.LittleEndian, &target.id)
							binary.Write(outbuf, binary.LittleEndian, &target.pos.X)
							binary.Write(outbuf, binary.LittleEndian, &target.pos.Y)
						}
					}
				}

				binary.Write(countbuf, binary.LittleEndian, &numPlayers)
				writeToPlayer(player, CMD_UPDATE, append(countbuf.Bytes(), outbuf.Bytes()...))
				//buf := fmt.Sprintf("b/sec: %v", len(outbuf.Bytes())*15)
				//fmt.Println(buf)
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
				doLog(true, "Unable to keep up: took: %v", took)
			}

		}
	}()

	if gTestMode {
		for i := 0; i < 1500; i++ {
			startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(5000)),
				Y: uint32(int(xyHalf) + rand.Intn(5000))}
			player := &playerData{id: makePlayerID(), pos: startLoc, area: &testArea, bot: true}
			pListLock.Lock()
			playerList[player.id] = player
			addPlayerToWorld(player.area, startLoc, player)
			pListLock.Unlock()
		}
	}
}

const chunkDiv = 256
const numChunks = 3

func addPlayerToWorld(area *areaData, pos XY, player *playerData) {
	if area == nil {
		return
	}
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	/* Create chunk if needed */
	chunk := area.chunks[chunkPos]
	if chunk == nil {
		area.chunks[chunkPos] = &chunkData{}
		doLog(true, "Created chunk: %v,%v", chunkPos.X, chunkPos.Y)
	}

	/* Create entry */
	area.chunks[chunkPos].players =
		append(area.chunks[chunkPos].players,
			player)
}

func removePlayerWorld(area *areaData, pos XY, player *playerData) {
	if area == nil {
		return
	}

	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	chunkPlayers := area.chunks[chunkPos].players
	var deleteme int = -1
	var numPlayers = len(chunkPlayers) - 1
	for t, target := range chunkPlayers {
		if target.id == player.id {
			deleteme = t
			break
		}
	}
	area.chunks[chunkPos].players[deleteme] =
		area.chunks[chunkPos].players[numPlayers]

	area.chunks[chunkPos].players = chunkPlayers[:numPlayers]
}

func movePlayer(area *areaData, pos XY, player *playerData) {
	removePlayerWorld(area, player.pos, player)
	addPlayerToWorld(area, pos, player)
	player.pos = pos
}
