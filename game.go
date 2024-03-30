package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/xy"
)

var processLock sync.RWMutex

const playerSize = 24
const grace = 10
const searchSize = 2

func getClosestTarget(player *playerData) *playerData {
	var closestTarget *playerData = nil
	var closestDist float64 = 10000

	// Check surrounding area for collisions
	for x := -searchSize; x <= searchSize; x++ {
		for y := -searchSize; y <= searchSize; y++ {
			//Get chunk
			intPos := floorXY(&player.pos)
			chunkPos := XY{X: uint32(int(intPos.X/chunkDiv) + x),
				Y: uint32(int(intPos.Y/chunkDiv) + y)}
			chunk := player.area.Chunks[chunkPos]
			if chunk == nil {
				continue
			}

			//Find targets
			for t, target := range chunk.players {
				if !target.VALID {
					continue
				}
				//Skip creatures
				if target.creatureData != nil {
					continue
				}
				dist := distanceFloat(player.pos, target.pos)
				if dist < closestDist {
					closestDist = dist
					closestTarget = chunk.players[t]
				}
			}
		}
	}

	return closestTarget
}

func dirTo(player, target *playerData) DIR {
	p1 := geom.Coord{float64(player.pos.X), float64(player.pos.Y), 0}
	p2 := geom.Coord{float64(target.pos.X), float64(target.pos.Y), 0}
	angle := xy.Angle(p1, p2)

	return radiansToDirection(angle)
}

const twoPi = math.Pi * 2.0
const offset = math.Pi / 2.0

func radiansToDirection(in float64) DIR {
	defer reportPanic("radiansToDirection")

	rads := math.Mod(in+offset, twoPi)
	normal := (rads / twoPi) * 100.0

	//Lame hack, TODO FIXME
	if normal < 0 {
		normal = 87.5
	}
	amount := int(math.Round(normal / 12.5))
	return DIR(amount)
}

func movePlayer(player *playerData, test bool) bool {

	if hasEffects(player, EFFECT_INJURED) {
		return false
	}

	var newPos XYf32 = player.pos
	if player.creatureData == nil {
		newPos = moveDir(player.pos, player.moveDir, walkSpeed)
	} else {
		newPos = moveDir(player.pos, player.moveDir, walkSpeed/3)
	}

	// Check surrounding area for collisions
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {

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

				if dist < playerSize {
					addTarget(player, target, 0, 0)
					return false
				}
			}

			//Find player-creature collisions
			for _, target := range chunk.creatrues {
				if player.creatureData != nil {
					if player.creatureData.id.UID == target.creatureData.id.UID {
						//skip self
						continue
					}
				}
				dist := distanceFloat(target.pos, newPos)

				if dist < playerSize {
					addTarget(player, target, 0, 0)
					return false
				}
			}

			//Find world object collisions
			for _, target := range chunk.WorldObjects {
				if target.ID.Section != 3 {
					continue
				}
				dist := distanceInt(target.Pos, floorXY(&newPos))
				if dist < 48 {
					return false
				}
			}
		}
	}
	// Otherwise, move player
	movePlayerChunk(player.area, newPos, player)

	return true
}

func affect(player *playerData) {

	//Reset attack effect, so animation stops if needed
	if hasEffects(player, EFFECT_ATTACK) {
		removeEffect(player, EFFECT_ATTACK)
	}

	//Check all our targets
	for p, t := range player.targets {
		//If either player has left the game...
		//Or we are injured, or the target is too far away... remove effects
		if !player.VALID || !t.target.VALID ||
			hasEffects(player, EFFECT_INJURED) ||
			distanceFloat(player.pos, t.target.pos) > playerSize+grace {
			removeTarget(player, t.target)
			continue
		}

		if player.mode == PMODE_ATTACK { //ATTACKING

			//If the player is not injured, damage them... if correct interval
			if !hasEffects(t.target, EFFECT_INJURED) {
				if gameTick%6 == 0 {
					if t.target.creatureData != nil {
						t.target.health -= 24
					} else {
						t.target.health -= 6
					}
				}

				//If their health goes to 0, set as injured and stop current movement
				//Make health negative, so they can't be revived instantly
				if t.target.health < 1 {
					setEffect(t.target, EFFECT_INJURED)
					t.target.dir = DIR_NONE
					if t.target.creatureData == nil {
						send_chat(fmt.Sprintf("%v is injured!", t.target.name))
					}
					t.target.health -= 50

				} else {
					//Otherwise, start our attack animation
					setEffect(player, EFFECT_ATTACK)
				}
				break
			}

		} else if player.mode == PMODE_HEAL { //HEALING

			if t.target.creatureData != nil {
				continue
			}

			//If the target is injured, but their health is now above zero... Remove injured effect
			if hasEffects(t.target, EFFECT_INJURED) && t.target.health > 0 {
				removeEffect(t.target, EFFECT_INJURED)
			}
			//If their health isn't full yet
			if t.target.health < 100 {
				//Increase health every other tick
				if gameTick%2 == 0 {
					t.target.health++
				}

				player.targets[p].selfEffects = EFFECT_HEALER
				player.targets[p].targetEffects = EFFECT_HEAL
				setEffect(player, EFFECT_HEALER)
				setEffect(t.target, EFFECT_HEAL)

				//Stop here, no multi-target
				break
			} else {

				//Target's health is full, remove healing effect.
				removeTarget(player, t.target)
			}
		}
	}
}

// Adds to list, applies effects
// Does not add to list (or apply effects) if already in list.
func addTarget(player, newTarget *playerData, selfEffects, targetEffects EFF) {
	found := false
	for _, t := range player.targets {
		if t.target.id == newTarget.id {
			return
		}
		found = true
	}
	if !found {
		setEffect(player, selfEffects)
		setEffect(newTarget, targetEffects)
		player.targets = append(player.targets,
			&targetingData{target: newTarget, selfEffects: selfEffects, targetEffects: targetEffects})
		player.numTargets++
	}
}

// Removes target and removes effects
func removeTarget(player, removeTarget *playerData) {

	for i := 0; i < player.numTargets; i++ {
		if player.targets[i].target.id == removeTarget.id {
			removeEffect(player, player.targets[i].selfEffects)
			removeEffect(player.targets[i].target, player.targets[i].targetEffects)

			if player.numTargets == 1 {
				player.targets = []*targetingData{}
				player.numTargets = 0
				return
			}
			player.targets[i] = player.targets[player.numTargets-1]
			player.targets = player.targets[:player.numTargets-1]
			player.numTargets--
			break
		}
	}
}

func moveCreature(player *playerData) DIR {
	//Make sure this is a valid player
	if player.creatureData != nil && player.VALID {

		if player.creatureData.mode == CRE_ATTACK {
			target := getClosestTarget(player)

			if target != nil {
				return dirTo(player, target)
			}
		}

	}

	return DIR_NONE
}

var gameTick uint64 = 1

func processGame() {

	go func() {
		//defer reportPanic("processGame goroutine")
		time.Sleep(time.Second)

		//sized wait group, number of available threads
		wg := sizedwaitgroup.New(runtime.NumCPU())

		//Loop forever
		for {
			gameTick++
			loopStart := time.Now()

			var outsize atomic.Uint32
			processLock.Lock()
			if numPlayers > 0 {

				//Move player
				for _, player := range playerList {
					if player.health < 100 && player.health > 0 {
						if gameTick%30 == 0 {
							player.health++
						}
					}
					if player.moveDir != DIR_NONE {
						if int(gameTick)-int(player.lastDirUpdate) > lagThresh {
							player.moveDir = DIR_NONE
						}
						movePlayer(player, false)
					}
					affect(player)
				}

				//Move creature
				for _, area := range areaList {
					for _, chunk := range area.Chunks {
						for _, creature := range chunk.creatrues {
							if creature.creatureData.id.Section == 7 && //Creatures
								creature.creatureData.id.Num == 0 { //Zombie

								if creature.health < 100 { //Passive heal
									if gameTick%15 == 0 {
										creature.health++

										//If the creature is injured, but their health is now above zero... Remove injured effect
										if hasEffects(creature, EFFECT_INJURED) && creature.health > 0 {
											removeEffect(creature, EFFECT_INJURED)
										}
									}
								}
							}
							creature.moveDir = moveCreature(creature)
							if creature.moveDir != DIR_NONE {
								creature.dir = creature.moveDir
							}

							if creature.dir != DIR_NONE {
								movePlayer(creature, false)
							}
							affect(creature)
						}
					}
				}

				//Serialize data for transfer / cache
				//THREADED
				for _, player := range playerList {

					wg.Add()
					go func(player *playerData) {
						var playerBytes, objectBytes, creatureBytes []byte
						var playerRecords, objectRecords, creatureRecords uint8

						playerBuf := bytes.NewBuffer(playerBytes)
						objectBuf := bytes.NewBuffer(objectBytes)
						creatureBuf := bytes.NewBuffer(creatureBytes)

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

								var pBytes, oBytes, cBytes []byte
								pBuf := bytes.NewBuffer(pBytes)
								oBuf := bytes.NewBuffer(oBytes)
								cBuf := bytes.NewBuffer(cBytes)

								//PLAYERS
								//Use cache if found (chunk that two players can see)
								if chunk.pCacheTick == gameTick {
									pBuf.Write(chunk.playerCache)
									playerRecords += chunk.numPlayers
								} else {
									//Write players
									for _, target := range chunk.players {

										//17 bytes with header
										nx := uint32(xyCenter - int(target.pos.X))
										ny := uint32(xyCenter - int(target.pos.Y))
										binary.Write(pBuf, binary.LittleEndian, &target.id)
										binary.Write(pBuf, binary.LittleEndian, &nx)
										binary.Write(pBuf, binary.LittleEndian, &ny)
										binary.Write(pBuf, binary.LittleEndian, &target.dir)
										binary.Write(pBuf, binary.LittleEndian, &target.health)
										binary.Write(pBuf, binary.LittleEndian, &target.effects)
										playerRecords++
									}
									chunk.playerCache = pBuf.Bytes()
									chunk.pCacheTick = gameTick
								}
								playerBuf.Write(pBuf.Bytes())

								/* WORLD OBJECTS */
								/* Check if player needs this data or not, static objects */
								if player.visCache[chunkPos] == nil {
									addVis(player, chunkPos)

									//Use cache if found
									if chunk.hasOcache {
										oBuf.Write(chunk.objectCache)
										objectRecords += chunk.numWorldObjects
									} else {
										for _, obj := range chunk.WorldObjects {

											//12 bytes with header
											binary.Write(oBuf, binary.LittleEndian, obj.ID.Section)
											binary.Write(oBuf, binary.LittleEndian, obj.ID.Num)
											binary.Write(oBuf, binary.LittleEndian, obj.ID.Sprite)
											binary.Write(oBuf, binary.LittleEndian, obj.Pos.X)
											binary.Write(oBuf, binary.LittleEndian, obj.Pos.Y)
											objectRecords++
										}
										chunk.objectCache = oBuf.Bytes()
										chunk.hasOcache = true
									}
									objectBuf.Write(oBuf.Bytes())
								}

								/* CREATURES */
								//Use cache if found
								if uint64(chunk.cCacheTick) == gameTick {
									cBuf.Write(chunk.creatureCache)
								} else {
									for _, cre := range chunk.creatrues {
										//19 bytes with header
										nx := uint32(xyCenter - int(cre.pos.X))
										ny := uint32(xyCenter - int(cre.pos.Y))
										binary.Write(cBuf, binary.LittleEndian, &cre.creatureData.id.UID)
										binary.Write(cBuf, binary.LittleEndian, &cre.creatureData.id.Section)
										binary.Write(cBuf, binary.LittleEndian, &cre.creatureData.id.Num)
										binary.Write(cBuf, binary.LittleEndian, &nx)
										binary.Write(cBuf, binary.LittleEndian, &ny)
										binary.Write(cBuf, binary.LittleEndian, &cre.dir)
										binary.Write(cBuf, binary.LittleEndian, &cre.health)
										binary.Write(cBuf, binary.LittleEndian, &cre.effects)
									}
									chunk.creatureCache = cBuf.Bytes()
									chunk.cCacheTick = gameTick
								}
								creatureRecords += chunk.numCreatures
								creatureBuf.Write(cBuf.Bytes())

							}
						}

						// Write size headers
						var pcountBytes, ocountBytes, cCountBuf []byte
						pcountBuf := bytes.NewBuffer(pcountBytes)
						ocountBuf := bytes.NewBuffer(ocountBytes)
						ccountBuf := bytes.NewBuffer(cCountBuf)
						binary.Write(pcountBuf, binary.LittleEndian, &playerRecords)
						binary.Write(ocountBuf, binary.LittleEndian, &objectRecords)
						binary.Write(ccountBuf, binary.LittleEndian, &creatureRecords)

						//Combine everything.
						var outbytes []byte
						outbuf := bytes.NewBuffer(outbytes)
						outbuf.Write(pcountBuf.Bytes())
						outbuf.Write(playerBuf.Bytes())
						outbuf.Write(ocountBuf.Bytes())
						outbuf.Write(objectBuf.Bytes())
						outbuf.Write(ccountBuf.Bytes())
						outbuf.Write(creatureBuf.Bytes())

						outsize.Add(uint32(outbuf.Len()))
						writeToPlayer(player, CMD_WorldUpdate, outbuf.Bytes())

						wg.Done()
					}(player)

				}
				wg.Wait()

				//Bandwidth use
				if gameTick%150 == 0 {
					//Show bandwidth use
					if gTestMode && numConnections.Load() > 0 {
						fmt.Printf("Out: %0.2f mbit\n", float32(outsize.Load())*15.0/1024.0/1024.0)
					}
				}
			}
			processLock.Unlock()

			//Calculate remaining frame time
			took := time.Since(loopStart)
			remaining := (time.Nanosecond * FrameSpeedNS) - took

			//Sleep if there is remaining frame time
			if remaining > 0 {
				time.Sleep(remaining)

				if gTestMode {
					//Log frame time
					if gameTick%150 == 0 && numConnections.Load() > 0 {
						fmt.Printf("took: %v\n", took.Round(time.Millisecond))
					}
				}

			} else {
				//Log we are slower than real-time
				doLog(true, "Tick: %v: Unable to keep up: took: %v", gameTick, took.Round(time.Millisecond))
			}

		}
	}()

	/* Spawn players for test mode */
	if gTestMode {
		processLock.Lock()

		testPlayers := 1000
		space := testPlayers * 5
		hSpace := space / 2

		for i := 0; i < testPlayers; i++ {
			startLoc := XYf32{X: float32(hSpace - rand.Intn(space)),
				Y: float32(hSpace - rand.Intn(space))}
			pid := makePlayerID()
			player := &playerData{
				id: pid, name: fmt.Sprintf("Player-%v", pid), pos: startLoc, area: areaList[0],
				health: 100, dir: DIR_N, moveDir: DIR_NONE, lastDirUpdate: gameTick + 9000, VALID: true, visCache: make(map[XY]*visCacheData)}

			for !movePlayer(player, true) {
				tryPos := XYf32{X: float32(halfArea - rand.Intn(spawnArea)),
					Y: float32(halfArea - rand.Intn(spawnArea))}
				movePlayerChunk(player.area, tryPos, player)
				fmt.Println("Spawn blocked... Trying again.")
			}

			playerListLock.Lock()
			addPlayerToWorld(player.area, startLoc, player)
			playerListLock.Unlock()
		}
		processLock.Unlock()
	}

	go func() {
		for {
			time.Sleep(time.Minute)
			processLock.Lock()
			for _, area := range areaList {
				for c, chunk := range area.Chunks {
					if chunk.pCacheTick < gameTick {
						area.Chunks[c].playerCache = []byte{}
					}
				}
			}
			processLock.Unlock()

		}
	}()
}

func addVis(player *playerData, pos XY) {
	if player.visCache[pos] == nil {
		player.visCache[pos] = &visCacheData{pos: pos, lastSaw: gameTick}
		player.numVis++
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
	if player.creatureData != nil {
		chunk.creatrues = append(chunk.creatrues, player)
		chunk.numCreatures++
	} else {
		chunk.players = append(chunk.players, player)
		chunk.numPlayers++
	}
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
	chunk := area.Chunks[chunkPos]
	if chunk == nil {
		return
	}

	if player.creatureData != nil {
		for i := 0; i < int(chunk.numCreatures); i++ {
			if chunk.creatrues[i].creatureData.id.UID == player.creatureData.id.UID {
				if chunk.numCreatures == 1 {
					chunk.creatrues = []*playerData{}
					chunk.numCreatures = 0
					return
				}
				chunk.creatrues[i] = chunk.creatrues[chunk.numCreatures-1]
				chunk.creatrues = chunk.creatrues[:chunk.numCreatures-1]
				chunk.numCreatures--
				break
			}
		}
	} else {
		for i := 0; i < int(chunk.numPlayers); i++ {
			if chunk.players[i].id == player.id {
				if chunk.numPlayers == 1 {
					chunk.players = []*playerData{}
					chunk.numPlayers = 0
					return
				}
				chunk.players[i] = chunk.players[chunk.numPlayers-1]
				chunk.players = chunk.players[:chunk.numPlayers-1]
				chunk.numPlayers--
				break
			}
		}
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
	chunk.numWorldObjects++

	//Remove byte caches
	chunk.objectCache = []byte{}
	chunk.hasOcache = false

	removeVisCache(area, pos)
}

func removeVisCache(area *areaData, pos XY) {
	chunkPos := XY{X: uint32(int(pos.X / chunkDiv)),
		Y: uint32(int(pos.Y / chunkDiv))}

	//Remove from visCache from relevant players
	for _, player := range playerList {
		if player.area.ID != area.ID {
			continue
		}

		if player.visCache[chunkPos] != nil {
			delete(player.visCache, chunkPos)
			doLog(true, "addWorldObject: Removed from visCache for player %v", player.name)
		}
	}
}

func removeWorldObject(area *areaData, pos XY, id IID) {
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

	for i := 0; i < int(chunk.numWorldObjects); i++ {
		if samePos(chunk.WorldObjects[i].Pos, pos) && sameIID(chunk.WorldObjects[i].ID, id) {
			removeVisCache(area, pos)

			if chunk.numWorldObjects == 1 {
				chunk.WorldObjects = []*worldObject{}
				chunk.numWorldObjects = 0
				chunk.objectCache = []byte{}
				chunk.hasOcache = false
				return
			}

			chunk.WorldObjects[i] = chunk.WorldObjects[chunk.numWorldObjects-1]
			chunk.WorldObjects = chunk.WorldObjects[:chunk.numWorldObjects-1]
			chunk.numWorldObjects--
			chunk.objectCache = []byte{}
			chunk.hasOcache = false
			break
		}
	}
}

func moveWorldObject(area *areaData, pos XY, wObject *worldObject) {
	defer reportPanic("moveWorldObject")

	//Sanity check
	if wObject == nil || area == nil {
		return
	}

	//Remove player from old chunk
	removeWorldObject(area, wObject.Pos, wObject.ID)

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
