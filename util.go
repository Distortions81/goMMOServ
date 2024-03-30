package main

import (
	"bytes"
	"compress/zlib"
	"image"
	"io"
	"math"
	"sync"
)

func samePos(a, b XY) bool {
	if a.X == b.X && a.Y == b.Y {
		return true
	}
	return false
}

func samePosf(a, b XYf32) bool {
	if a.X == b.X && a.Y == b.Y {
		return true
	}
	return false
}

func moveDir(input XYf32, dir DIR, speed float32) XYf32 {

	pos := input

	switch dir {
	case DIR_N:
		pos.Y += speed
	case DIR_NE:
		pos.Y += speed * diagSpeed
		pos.X -= speed * diagSpeed
	case DIR_E:
		pos.X -= speed
	case DIR_SE:
		pos.X -= speed * diagSpeed
		pos.Y -= speed * diagSpeed
	case DIR_S:
		pos.Y -= speed
	case DIR_SW:
		pos.Y -= speed * diagSpeed
		pos.X += speed * diagSpeed
	case DIR_W:
		pos.X += speed
	case DIR_NW:
		pos.Y += speed * diagSpeed
		pos.X += speed * diagSpeed
	}
	return pos
}

func floorXY(input *XYf32) XY {
	return XY{X: uint32(xyCenter - int(input.X)), Y: uint32(xyCenter - int(input.Y))}
}

func floatXY(input *XY) XYf32 {
	return XYf32{X: float32(xyCenter + int(input.X)), Y: float32(xyCenter + int(input.Y))}
}

func distanceFloat(a, b XYf32) float64 {

	dx := a.X - b.X
	dy := a.Y - b.Y

	return math.Sqrt(float64(dx*dx + dy*dy))
}

func sameIID(a, b IID) bool {
	if a.Section != b.Section {
		return false
	}
	if a.Num != b.Num {
		return false
	}
	return true
}

func distanceInt(a, b XY) float64 {

	dx := a.X - b.X
	dy := a.Y - b.Y

	return math.Sqrt(float64(dx*dx + dy*dy))
}

/* Generic unzip []byte */
func UncompressZip(data []byte) []byte {
	defer reportPanic("UncompressZip")

	b := bytes.NewReader(data)

	z, err := zlib.NewReader(b)
	if err != nil {
		doLog(true, err.Error())
		return nil
	}
	defer z.Close()

	p, err := io.ReadAll(z)
	if err != nil {
		return nil
	}
	return p
}

/* Generic zip []byte */
func CompressZip(data []byte) []byte {
	defer reportPanic("compressZip")

	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, zlib.BestSpeed)
	if err != nil {
		doLog(true, err.Error())
		return nil
	}
	w.Write(data)
	w.Close()
	return b.Bytes()
}

var playerTopID uint32
var playerTopIDLock sync.Mutex

var creatureTopID uint32
var creatureIDLock sync.Mutex

func makePlayerID() uint32 {
	playerTopIDLock.Lock()
	defer playerTopIDLock.Unlock()

	playerTopID++
	return playerTopID
}

func makeCreatureID() uint32 {
	creatureIDLock.Lock()
	defer creatureIDLock.Unlock()

	creatureTopID++
	return creatureTopID
}

// Check if a position is within a image.Rectangle
func PosWithinRect(pos XY, rect image.Rectangle, pad uint32) bool {
	defer reportPanic("PosWithinRect")

	if int(pos.X-pad) <= rect.Max.X && int(pos.X+pad) >= rect.Min.X {
		if int(pos.Y-pad) <= rect.Max.Y && int(pos.Y+pad) >= rect.Min.Y {
			return true
		}
	}
	return false
}

func justEnteredVis(player *playerData, pos XY) bool {
	for v, vis := range player.visCache {
		if vis.pos == pos {
			player.visCache[v].lastSaw = gameTick
			return false
		}
	}
	return true
}
