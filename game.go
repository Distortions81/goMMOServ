package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/remeh/sizedwaitgroup"
)

const FrameSpeedNS = 66666666

const chunkDiv = 128
const numChunks = 5

func processGame() {
	defer reportPanic("processGame")

	var gameTick uint64
	go func() {

		defer reportPanic("processGame goroutine")

		var pbuf, cbuf, obuf, ocbuf []byte
		//sized wait group, number of available threads
		wg := sizedwaitgroup.New(runtime.NumCPU())

		//Loop forever
		for {
			gameTick++
			loopStart := time.Now()

			//Lock playerlist, read
			playerListLock.RLock()
			var outsize atomic.Uint32

			for _, player := range playerList {

				//Saninity check
				if player.conn == nil {
					continue
				}

				wg.Add()
				go func(player *playerData) {
					//Lock player
					player.plock.Lock()
					defer player.plock.Unlock()

					var numPlayers uint16
					var numObj uint16

					countbuf := bytes.NewBuffer(cbuf)
					playerBuf := bytes.NewBuffer(pbuf)

					objCountBuf := bytes.NewBuffer((ocbuf))
					objBuf := bytes.NewBuffer(obuf)

					//Search surrounding chunks
					for x := -numChunks; x < numChunks; x++ {
						for y := -numChunks; y < numChunks; y++ {
							chunkPos := XY{X: uint32(int(player.pos.X/chunkDiv) + x),
								Y: uint32(int(player.pos.Y/chunkDiv) + y)}
							chunk := player.area.Chunks[chunkPos]
							if chunk == nil {
								continue
							}

							//Lock chunk
							chunk.chunkLock.Lock()
							for _, target := range chunk.players {

								//Serialize data
								binary.Write(playerBuf, binary.LittleEndian, &target.id)
								binary.Write(playerBuf, binary.LittleEndian, &target.pos.X)
								binary.Write(playerBuf, binary.LittleEndian, &target.pos.Y)

								//Eventually move me to an event
								binary.Write(playerBuf, binary.LittleEndian, &target.health)

								numPlayers++
							}
							//Tally players, needed for header

							//Tally output
							outsize.Add(uint32(numPlayers) * 104)

							//Write world objects

							for _, obj := range chunk.WorldObjects {
								binary.Write(objBuf, binary.LittleEndian, &obj.ItemId)
								binary.Write(objBuf, binary.LittleEndian, &obj.Pos.X)
								binary.Write(objBuf, binary.LittleEndian, &obj.Pos.Y)

								numObj++
							}

							// Tally output
							outsize.Add(uint32(numObj) * 96)

							//Unlock chunk
							chunk.chunkLock.Unlock()
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

			//Unlock player list
			playerListLock.RUnlock()

			//Calculate remaining frame time
			took := time.Since(loopStart)
			remaining := (time.Nanosecond * FrameSpeedNS) - took

			//Show bandwidth use
			if gTestMode && gameTick%450 == 0 && getNumberConnections() > 0 {
				fmt.Printf("Out: %vkbit\n", outsize.Load()*15/1024)
			}

			//Sleep if there is remaining frame time
			if remaining > 0 {
				time.Sleep(remaining)

				if gTestMode {
					//Log frame time
					if gameTick%450 == 0 && getNumberConnections() > 0 {
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
		for i := 0; i < 5000; i++ {
			startLoc := XY{X: uint32(int(xyHalf) + rand.Intn(20000)),
				Y: uint32(int(xyHalf) + rand.Intn(20000))}
			player := &playerData{id: makePlayerID(), pos: startLoc, area: areaList[0], health: 100}
			playerListLock.Lock()
			playerList[player.id] = player
			addPlayerToWorld(player.area, startLoc, player)
			playerListLock.Unlock()
		}
	}
}

func addPlayerToWorld(area *areaData, pos XY, player *playerData) {
	defer reportPanic("addPlayerToWorld")

	//Sanity check
	if area == nil {
		return
	}
	//Calulate chunk pos
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	//Get chunk
	chunk := area.Chunks[chunkPos]

	//Create chunk if needed
	if chunk == nil {
		area.arealock.Lock()
		area.Chunks[chunkPos] = &chunkData{}
		area.arealock.Unlock()
		doLog(true, "Created chunk: %v,%v", chunkPos.X, chunkPos.Y)
	}

	/* Add player */
	area.Chunks[chunkPos].chunkLock.Lock()
	area.Chunks[chunkPos].players =
		append(area.Chunks[chunkPos].players,
			player)
	area.Chunks[chunkPos].chunkLock.Unlock()
}

func removePlayerWorld(area *areaData, pos XY, player *playerData) {
	defer reportPanic("removePlayerWorld")

	//Sanity check
	if player == nil || area == nil {
		return
	}

	//Calc chunk pos
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	//Lock chunk
	area.Chunks[chunkPos].chunkLock.Lock()
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
	//Unlock chunk
	area.Chunks[chunkPos].chunkLock.Unlock()
}

func movePlayer(area *areaData, pos XY, player *playerData) {
	defer reportPanic("movePlayer")

	//Remove player from old chunk
	removePlayerWorld(area, player.pos, player)

	//Add player to new chunk
	addPlayerToWorld(area, pos, player)

	//Update player position
	player.pos = pos
}

func addWorldObject(area *areaData, pos XY, wObject *worldObject) {
	defer reportPanic("addWorldObject")

	//Sanity check
	if area == nil {
		return
	}
	//Calulate chunk pos
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	//Get chunk
	chunk := area.Chunks[chunkPos]

	//Create chunk if needed
	if chunk == nil {
		area.arealock.Lock()
		area.Chunks[chunkPos] = &chunkData{}
		area.arealock.Unlock()
		doLog(true, "Created chunk: %v,%v", chunkPos.X, chunkPos.Y)
	}

	/* Add object */
	area.Chunks[chunkPos].chunkLock.Lock()
	area.Chunks[chunkPos].WorldObjects = append(area.Chunks[chunkPos].WorldObjects, wObject)
	area.Chunks[chunkPos].chunkLock.Unlock()
}

func removeWorldObject(area *areaData, pos XY, wObject *worldObject) {
	defer reportPanic("removePlayerWorld")

	//Sanity check
	if wObject == nil || area == nil {
		return
	}

	//Calc chunk pos
	chunkPos := XY{X: pos.X / chunkDiv, Y: pos.Y / chunkDiv}

	//Lock chunk
	area.Chunks[chunkPos].chunkLock.Lock()
	//Get players in chunk
	chunkObjects := area.Chunks[chunkPos].WorldObjects

	//Find player
	var deleteme int = -1
	var numObjs = len(chunkObjects) - 1
	for t, target := range chunkObjects {
		if target.uid == wObject.uid {
			deleteme = t
			break
		}
	}

	//Sanity check
	if deleteme >= 0 {
		//Fast, non-order-preserving delete player from chunk
		area.Chunks[chunkPos].WorldObjects[deleteme] =
			area.Chunks[chunkPos].WorldObjects[numObjs]
		area.Chunks[chunkPos].WorldObjects = chunkObjects[:numObjs]
	}
	//Unlock chunk
	area.Chunks[chunkPos].chunkLock.Unlock()
}

func moveWorldObject(area *areaData, pos XY, wObject *worldObject) {
	defer reportPanic("movePlayer")

	//Remove player from old chunk
	removeWorldObject(area, wObject.Pos, wObject)

	//Add player to new chunk
	addWorldObject(area, pos, wObject)

	//Update player position
	wObject.Pos = pos
}
