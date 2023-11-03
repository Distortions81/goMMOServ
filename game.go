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
	FrameSpeedNS = 66666666
	chunkDiv     = 128
	searchChunks = 6
	lagThresh    = 8
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

			//Find player collisions
			for _, target := range chunk.players {

				if target.id == player.id {
					//Skip self
					continue
				}
				dist := distanceFloat(target.pos, newPos)

				if dist < 10 {
					//fmt.Printf("Items inside each other! %v and %v (%v p)\n", target.id, player.id, dist)
					newPos.X += 32
					newPos.Y += 32

					break
				} else if dist < 24 {
					if player.mode == PMODE_ATTACK {
						if target.health > 0 {
							target.health--
						}
					} else if player.mode == PMODE_HEAL {
						if target.health < 100 {
							target.health++
						}
					}
					return
				}

			}

			//Find world object collisions
			for _, target := range chunk.WorldObjects {

				dist := distanceFloat(floatXY(&target.Pos), newPos)

				if dist < 48 {
					return
				}

			}

		}
	}

	// Otherwise, move player
	movePlayerChunk(player.area, newPos, player)
}

var gameTick uint64 = 1

func processGame() {

	go func() {
		defer reportPanic("processGame goroutine")
		time.Sleep(time.Second)

		var playerBytes, pCountBytes, objBytes, objCountBytes []byte

		//sized wait group, number of available threads
		wg := sizedwaitgroup.New(runtime.NumCPU())

		//Loop forever
		for {
			gameTick++
			loopStart := time.Now()

			var outsize atomic.Uint32
			processLock.Lock()

			//Move player
			for _, player := range playerList {
				if player.dir != DIR_NONE {
					if int(gameTick)-int(player.lastDirUpdate) > lagThresh {
						player.dir = DIR_NONE
					}
					movePlayer(player)
				}
			}
			for _, player := range playerList {

				wg.Add()
				go func(player *playerData) {

					var TnumPlayers, TnumObj uint16

					pCountBuf := bytes.NewBuffer(pCountBytes)
					playerBuf := bytes.NewBuffer(playerBytes)

					oCountBuf := bytes.NewBuffer((objCountBytes))
					objBuf := bytes.NewBuffer(objBytes)

					//Search surrounding chunks
					for x := -searchChunks; x < searchChunks; x++ {
						for y := -searchChunks; y < searchChunks; y++ {

							var oCount, pCount uint16

							//Calc chunk pos
							intPos := floorXY(&player.pos)
							chunkPos := XY{X: uint32(int(intPos.X/chunkDiv) + x), Y: uint32(int(intPos.Y/chunkDiv) + y)}
							chunk := player.area.Chunks[chunkPos]

							if chunk == nil {
								continue
							}

							chunk.chunkLock.RLock()
							if chunk.bufferFrame == gameTick {
								playerBuf.Write(chunk.playerBuffer)
								objBuf.Write(chunk.objBuffer)

								TnumPlayers += chunk.pBufCount
								TnumObj += chunk.oBufCount
								chunk.chunkLock.RUnlock()

								continue
							}
							chunk.chunkLock.RUnlock()

							var cBytes, pBytes []byte
							oBuf := bytes.NewBuffer(cBytes)
							pBuf := bytes.NewBuffer(pBytes)

							//Write players
							for _, target := range chunk.players {

								//120 bytes with header
								nx := uint32(xyCenter - int(target.pos.X))
								ny := uint32(xyCenter - int(target.pos.Y))
								binary.Write(pBuf, binary.LittleEndian, &target.id)
								binary.Write(pBuf, binary.LittleEndian, &nx)
								binary.Write(pBuf, binary.LittleEndian, &ny)
								binary.Write(pBuf, binary.LittleEndian, &target.health)

								//Tally players, needed for header
								pCount++
							}
							TnumPlayers += pCount

							//Write dynamic world objects
							for _, obj := range chunk.WorldObjects {

								//112 bytes with header
								binary.Write(oBuf, binary.LittleEndian, &obj.ItemId)
								binary.Write(oBuf, binary.LittleEndian, &obj.Pos.X)
								binary.Write(oBuf, binary.LittleEndian, &obj.Pos.Y)

								oCount++
							}

							TnumObj += oCount

							chunk.chunkLock.Lock()
							chunk.playerBuffer = pBuf.Bytes()
							chunk.objBuffer = oBuf.Bytes()

							playerBuf.Write(chunk.playerBuffer)
							objBuf.Write(chunk.objBuffer)

							outsize.Add(uint32(120 * TnumPlayers))
							outsize.Add(uint32(112 * TnumObj))

							chunk.pBufCount = pCount
							chunk.oBufCount = oCount

							chunk.bufferFrame = gameTick
							chunk.cleanme = true
							chunk.chunkLock.Unlock()
						}
					}

					//Write header
					binary.Write(pCountBuf, binary.LittleEndian, &TnumPlayers)
					binary.Write(oCountBuf, binary.LittleEndian, &TnumObj)

					//Write the whole thing
					playerOut := append(pCountBuf.Bytes(), playerBuf.Bytes()...)
					objOut := append(oCountBuf.Bytes(), objBuf.Bytes()...)

					writeToPlayer(player, CMD_WorldUpdate, append(playerOut, objOut...))

					wg.Done()
				}(player)

			}
			wg.Wait()

			// Remove caches of unoccupied areas
			if gameTick%900 == 0 {
				for _, area := range areaList {
					for _, chunk := range area.Chunks {
						chunk.objBuffer = []byte{}
						chunk.playerBuffer = []byte{}
					}
				}
			}

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
				doLog(true, "%v: Unable to keep up: took: %v", gameTick, took.Round(time.Millisecond))
			}

		}
	}()

	/* Spawn players for test mode */
	if gTestMode {
		processLock.Lock()

		testPlayers := 30000
		space := testPlayers * 2
		hSpace := space / 2

		for i := 0; i < testPlayers; i++ {
			startLoc := XYf32{X: float32(hSpace - rand.Intn(space)),
				Y: float32(hSpace - rand.Intn(space))}
			player := &playerData{
				id: makePlayerID(), pos: startLoc, area: areaList[0],
				health: 100, dir: DIR_N, lastDirUpdate: gameTick + 9000}
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
