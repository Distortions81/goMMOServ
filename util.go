package main

import (
	"bytes"
	"compress/zlib"
	"image"
	"io"
	"math"
	"sync"
)

const diagSpeed = 0.70710678118
const walkSpeed = 8

func moveDir(input XYf32, dir DIR) XYf32 {

	pos := input

	switch dir {
	case DIR_N:
		pos.Y += walkSpeed
	case DIR_NE:
		pos.Y += walkSpeed * diagSpeed
		pos.X -= walkSpeed * diagSpeed
	case DIR_E:
		pos.X -= walkSpeed
	case DIR_SE:
		pos.X -= walkSpeed * diagSpeed
		pos.Y -= walkSpeed * diagSpeed
	case DIR_S:
		pos.Y -= walkSpeed
	case DIR_SW:
		pos.Y -= walkSpeed * diagSpeed
		pos.X += walkSpeed * diagSpeed
	case DIR_W:
		pos.X += walkSpeed
	case DIR_NW:
		pos.Y += walkSpeed * diagSpeed
		pos.X += walkSpeed * diagSpeed
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
	w, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
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

var objectTopID uint64
var objectTopIDLock sync.Mutex

func makePlayerID() uint32 {
	playerTopIDLock.Lock()
	defer playerTopIDLock.Unlock()

	playerTopID++
	return playerTopID
}

func makeObjectID() uint64 {
	objectTopIDLock.Lock()
	defer objectTopIDLock.Unlock()

	objectTopID++
	return objectTopID
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
