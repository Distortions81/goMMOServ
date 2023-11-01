package main

import (
	"bytes"
	"compress/zlib"
	"io"
	"math"
	"sync"
)

const diagSpeed = 0.70
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
	return XY{X: uint32(xyHalf - input.X), Y: uint32(xyHalf - input.Y)}
}

func floatXY(input *XY) XYf32 {
	return XYf32{X: float32(input.X - xyHalf), Y: float32(input.Y - xyHalf)}
}

func distanceFloat(a, b XYf32) float64 {

	dx := a.X - b.X
	dy := a.Y - b.Y

	return math.Sqrt(float64(dx*dx + dy*dy))
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
