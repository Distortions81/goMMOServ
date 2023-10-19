package main

import "time"

const FrameSpeedMS = 166

func processGame() {
	go func() {
		for {
			loopStart := time.Now()

			var output []byte
			for _, player := range playerList {
				bufID := uint32ToByteArray(player.id)
				bufXY := xyToByteArray(player.location.pos)
				output = append(output, bufID...)
				output = append(output, bufXY...)
			}
			for _, player := range playerList {
				writeToPlayer(player, CMD_UPDATE, output)
				doLog(true, "meep")
			}

			took := time.Since(loopStart)
			remaining := (time.Millisecond * FrameSpeedMS) - took

			if remaining > 0 { /*Kill remaining time*/
				time.Sleep(remaining)

			} else { /*We are lagging behind realtime*/
				time.Sleep(time.Millisecond)
				doLog(true, "Unable to keep up: took: %v", took)
			}
		}
	}()
}
