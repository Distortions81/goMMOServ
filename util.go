package main

import (
	"bytes"
	"compress/zlib"
	"io"
	"math"
	"sync"
)

const diagSpeed = 0.707

func moveDir(pos *XYf32, dir DIR) {

	switch dir {
	case DIR_N:
		pos.Y++
	case DIR_NE:
		pos.Y += diagSpeed
		pos.X -= diagSpeed
	case DIR_E:
		pos.X--
	case DIR_SE:
		pos.X -= diagSpeed
		pos.Y -= diagSpeed
	case DIR_S:
		pos.Y--
	case DIR_SW:
		pos.Y -= diagSpeed
		pos.X += diagSpeed
	case DIR_W:
		pos.X++
	case DIR_NW:
		pos.Y += diagSpeed
		pos.X += diagSpeed
	default:
		return
	}

}

func distance(a, b XY) float64 {

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
