package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/remeh/sizedwaitgroup"
)

const (
	FrameSpeedNS        = 66666666
	chunkDiv            = 128
	searchChunks        = 5
	lagThresh    uint64 = 8
)

var processLock sync.RWMutex

func movePlayer(player *playerData) {

	newPos := moveDir(player.pos, player.dir)

	// Check surrounding area for collisions
	for x := -1; x < 1; x++ {
		for y := -1; y < 1; y++ {

			//Get chunk
			intPos := floorXY(&player.pos)
			chunkPos := XY{X: uint32(int(intPos.X/chunkDiv) + x),
				Y: uint32(int(intPos.Y/chunkDiv) + y)}
			chunk := player.area.Chunks[chunkPos]
			if chunk == nil {
				continue
			}

			//Check chunk for collision
			for _, target := range chunk.players {

				if target.id == player.id {
					//Skip self
					continue
				}
				dist := distanceFloat(target.pos, newPos)

				if dist < 10 {
					fmt.Printf("Items inside each other! %v and %v (%v p)\n", target.id, player.id, dist)
					newPos.X += 24
					newPos.Y += 24

					return
				} else if dist < 24 {

					//Don't move, player is in our way
					fmt.Printf("BONK! #%v and #%v (%v p)\n", target.id, player.id, dist)
					return
				}

			}
		}
	}

	// Otherwise, move player
	movePlayerChunk(player.area, newPos, player)
}

var gameTick uint64

func processGame() {

	go func() {
		defer reportPanic("processGame goroutine")
		time.Sleep(time.Second)

		var playerBytes, countBytes, objectBytes, objectCountBytes []byte

		//sized wait group, number of available threads
		wg := sizedwaitgroup.New(runtime.NumCPU())

		//Loop forever
		for {
			gameTick++
			loopStart := time.Now()

			var outsize atomic.Uint32
			processLock.Lock()

			for _, player := range playerList {
				if player.dir != DIR_NONE {
					if gameTick-player.lastDirUpdate > lagThresh {
						player.dir = DIR_NONE
					}
					movePlayer(player)
				}
			}
			for _, player := range playerList {

				wg.Add()
				go func(player *playerData) {

					var numPlayers uint16
					var numObj uint16

					countbuf := bytes.NewBuffer(countBytes)
					playerBuf := bytes.NewBuffer(playerBytes)

					objCountBuf := bytes.NewBuffer((objectCountBytes))
					objBuf := bytes.NewBuffer(objectBytes)

					//Search surrounding chunks
					for x := -searchChunks; x < searchChunks; x++ {
						for y := -searchChunks; y < searchChunks; y++ {
							//Calc chunk pos
							intPos := floorXY(&player.pos)
							chunkPos := XY{X: uint32(int(intPos.X/chunkDiv) + x), Y: uint32(int(intPos.Y/chunkDiv) + y)}

							chunk := player.area.Chunks[chunkPos]

							if chunk == nil {
								continue
							}

							//Write players
							for _, target := range chunk.players {
								nx := uint32(xyHalf - int(target.pos.X))
								ny := uint32(xyHalf - int(target.pos.Y))
								binary.Write(playerBuf, binary.LittleEndian, &target.id)
								binary.Write(playerBuf, binary.LittleEndian, &nx)
								binary.Write(playerBuf, binary.LittleEndian, &ny)
								binary.Write(playerBuf, binary.LittleEndian, &target.health)

								//Tally players, needed for header
								numPlayers++
							}

							//Tally output
							outsize.Add(uint32(numPlayers) * 104)

							//Write dynamic world objects
							for _, obj := range chunk.WorldObjects {
								binary.Write(objBuf, binary.LittleEndian, &obj.ItemId)
								binary.Write(objBuf, binary.LittleEndian, &obj.Pos.X)
								binary.Write(objBuf, binary.LittleEndian, &obj.Pos.Y)

								numObj++
							}

							// Tally output
							outsize.Add(uint32(numObj) * 96)
						}
					}

					//Write header
					binary.Write(countbuf, binary.LittleEndian, &numPlayers)
					outsize.Add(16)
					binary.Write(objCountBuf, binary.LittleEndian, &numObj)
					outsize.Add(16)

					//Write the whole thing
					playerOut := append(countbuf.Bytes(), playerBuf.Bytes()...)
					objOut := append(objCountBuf.Bytes(), objBuf.Bytes()...)

					writeToPlayer(player, CMD_UPDATE, append(playerOut, objOut...))

					wg.Done()
				}(player)

			}
			wg.Wait()
			processLock.Unlock()

			//Calculate remaining frame time
			took := time.Since(loopStart)
			remaining := (time.Nanosecond * FrameSpeedNS) - took

			//Show bandwidth use
			if gTestMode && gameTick%450 == 0 && numConnections.Load() > 0 {
				fmt.Printf("Out: %v mbit\n", outsize.Load()*15/1024/1024)
			}

			//Sleep if there is remaining frame time
			if remaining > 0 {
				time.Sleep(remaining)

				if gTestMode {
					//Log frame time
					if gameTick%450 == 0 && numConnections.Load() > 0 {
						fmt.Printf("took: %v\n", took.Round(time.Millisecond))
					}
				}

			} else {
				/*Log we are slower than real-time*/
				doLog(true, "Unable to keep up: took: %v", took.Round(time.Millisecond))
			}

		}
	}()

	/* Spawn players for test mode */
	if gTestMode {
		processLock.Lock()
		for i := 0; i < 2500; i++ {
			startLoc := XYf32{X: float32(rand.Intn(20000)),
				Y: float32(rand.Intn(20000))}
			player := &playerData{id: makePlayerID(), pos: startLoc, area: areaList[0], health: 100}
			playerListLock.Lock()
			playerList = append(playerList, player)
			playerListLock.Unlock()
			addPlayerToWorld(player.area, startLoc, player)
		}
		processLock.Unlock()
	}
}

func addPlayerToWorld(area *areaData, pos XYf32, player *playerData) {
	defer reportPanic("addPlayerToWorld")

	//Sanity check
	if player == nil || area == nil {
		return
	}

	intPos := floorXY(&pos)

	//Get chunk
	chunk := getChunk(area, intPos)

	//Create chunk if needed
	if chunk == nil {
		chunk = assignChunk(area, intPos, &chunkData{})
	}

	/* Add player */
	chunk.players = append(chunk.players, player)
}

func removePlayerWorld(area *areaData, pos XYf32, player *playerData) {
	defer reportPanic("removePlayerWorld")

	//Sanity check
	if player == nil || area == nil {
		return
	}

	intPos := floorXY(&pos)

	//Calc chunk pos
	chunkPos := XY{X: uint32(intPos.X / chunkDiv), Y: uint32(intPos.Y / chunkDiv)}

	//Get players in chunk
	chunkPlayers := area.Chunks[chunkPos].players

	//Find player
	var deleteme int = -1
	var numPlayers = len(chunkPlayers) - 1
	for t, target := range chunkPlayers {
		if target.id == player.id {
			deleteme = t
			break
		}
	}

	//Sanity check
	if deleteme >= 0 {
		//Fast, non-order-preserving delete player from chunk
		area.Chunks[chunkPos].players[deleteme] =
			area.Chunks[chunkPos].players[numPlayers]
		area.Chunks[chunkPos].players = chunkPlayers[:numPlayers]
	}

}

func movePlayerChunk(area *areaData, newPos XYf32, player *playerData) {
	defer reportPanic("movePlayer")

	//Sanity check
	if player == nil || area == nil {
		return
	}

	//Remove player from old chunk
	removePlayerWorld(area, player.pos, player)

	//Add player to new chunk
	addPlayerToWorld(area, newPos, player)

	//Update player position
	player.pos = newPos
}

func addWorldObject(area *areaData, pos XY, wObject *worldObject) {
	defer reportPanic("addWorldObject")

	//Sanity check
	if area == nil || wObject == nil {
		return
	}

	//Get chunk
	chunk := getChunk(area, pos)

	//Create chunk if needed
	if chunk == nil {
		chunk = assignChunk(area, pos, &chunkData{})
	}

	/* Add object */
	chunk.WorldObjects = append(chunk.WorldObjects, wObject)
}

func removeWorldObject(area *areaData, pos XY, uid uint32) {
	defer reportPanic("removePlayerWorld")

	//Sanity check
	if area == nil {
		return
	}

	chunk := getChunk(area, pos)

	//Nothing here, exit
	if chunk == nil {
		return
	}

	//Find obj
	var deleteme int = -1
	var numObjs = len(chunk.WorldObjects) - 1
	for t, target := range chunk.WorldObjects {
		if target.uid == uid {
			deleteme = t
			break
		}
	}

	//If last object, just clear list
	if numObjs <= 0 {
		chunk.WorldObjects = []*worldObject{}
		return
	}

	//If found, delete
	if deleteme > -1 {

		//Fast, but does not preserve order
		chunk.WorldObjects[deleteme] =
			chunk.WorldObjects[numObjs]
		chunk.WorldObjects = chunk.WorldObjects[:numObjs]
	}
}

func moveWorldObject(area *areaData, pos XY, wObject *worldObject) {
	defer reportPanic("moveWorldObject")

	//Sanity check
	if wObject == nil || area == nil {
		return
	}

	//Remove player from old chunk
	removeWorldObject(area, wObject.Pos, wObject.uid)

	//Add player to new chunk
	addWorldObject(area, pos, wObject)

	//Update position
	wObject.Pos = pos
}

func getChunk(area *areaData, pos XY) *chunkData {
	defer reportPanic("getChunk")

	//Sanity check
	if area == nil {
		return nil
	}

	//Calc chunk pos
	chunkPos := XY{X: uint32(pos.X / chunkDiv), Y: uint32(pos.Y / chunkDiv)}

	area.areaLock.RLock()
	defer area.areaLock.RUnlock()

	return area.Chunks[chunkPos]
}

func assignChunk(area *areaData, pos XY, chunk *chunkData) *chunkData {
	defer reportPanic("assignChunk")

	//Sanity check
	if area == nil || chunk == nil {
		return nil
	}

	chunkPos := XY{X: uint32(pos.X / chunkDiv), Y: uint32(pos.Y / chunkDiv)}

	area.areaLock.Lock()
	defer area.areaLock.Unlock()

	area.Chunks[chunkPos] = chunk

	return chunk
}
