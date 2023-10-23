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

			pListLock.RLock()

			for _, player := range playerList {
				outbuf.Reset()

				superChunkPos := XY{X: player.pos.X / superChunkDiv, Y: player.pos.Y / superChunkDiv}
				chunkPos := XY{X: player.pos.X / chunkDiv, Y: player.pos.Y / chunkDiv}
				chunk := player.area.superChunks[superChunkPos].chunks[chunkPos]

				var numPlayers uint32 = uint32(len(chunk.players))

				binary.Write(outbuf, binary.LittleEndian, &numPlayers)
				for _, target := range chunk.players {
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
		for i := 0; i < 50000; i++ {
			startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(30000)),
				Y: uint32(int(xyHalf) + rand.Intn(30000))}
			player := &playerData{id: makePlayerID(), pos: startLoc, area: &testArea}
			pListLock.Lock()
			playerList[player.id] = player
			addPlayerToWorld(player.area, startLoc, player)
			pListLock.Unlock()
		}
	}
}

const chunkDiv = 128
const superChunkDiv = 128 * chunkDiv

func addPlayerToWorld(area *areaData, pos XY, player *playerData) {
	if area == nil {
		return
	}
	superChunkPos := XY{X: pos.X / superChunkDiv, Y: pos.Y / superChunkDiv}
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	superChunk := area.superChunks[superChunkPos]

	/* Create superchunk if needed */
	if superChunk == nil {
		area.superChunks[superChunkPos] = &superChunkData{chunks: make(map[XY]*chunkData)}
		doLog(true, "Created superChunk: %v,%v", superChunkPos.X, superChunkPos.Y)
	}

	/* Create chunk if needed */
	chunk := area.superChunks[superChunkPos].chunks[chunkPos]
	if chunk == nil {
		area.superChunks[superChunkPos].chunks[chunkPos] = &chunkData{}
		doLog(true, "Created chunk: %v,%v", chunkPos.X, chunkPos.Y)
	}

	/* Create entry */
	area.superChunks[superChunkPos].chunks[chunkPos].players =
		append(area.superChunks[superChunkPos].chunks[chunkPos].players,
			player)
}

func removePlayerWorld(area *areaData, pos XY, player *playerData) {
	if area == nil {
		return
	}

	superChunkPos := XY{X: pos.X / superChunkDiv, Y: pos.Y / superChunkDiv}
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	chunkPlayers := area.superChunks[superChunkPos].chunks[chunkPos].players
	var deleteme int = -1
	var numPlayers = len(chunkPlayers) - 1
	for t, target := range chunkPlayers {
		if target.id == player.id {
			deleteme = t
			break
		}
	}
	area.superChunks[superChunkPos].chunks[chunkPos].players[deleteme] =
		area.superChunks[superChunkPos].chunks[chunkPos].players[numPlayers]

	area.superChunks[superChunkPos].chunks[chunkPos].players = chunkPlayers[:numPlayers]
}

func movePlayer(area *areaData, pos XY, player *playerData) {
	removePlayerWorld(area, player.pos, player)
	addPlayerToWorld(area, pos, player)
	player.pos = pos
}
