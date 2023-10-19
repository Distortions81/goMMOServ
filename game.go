package main

import "time"

const FrameSpeedMS = 166

func processGame() {
	go func() {
		for {
			loopStart := time.Now()

			var buffer []byte
			for _, player := range playerList {
				buffer = append(buffer, xyToByteArray(player.location.pos)...)
			}
			for _, player := range playerList {
				writeToPlayer(player, CMD_UPDATE, buffer)
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
