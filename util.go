package main

import (
	"bytes"
	"compress/zlib"
	"io"
	"math"

	"github.com/sasha-s/go-deadlock"
)

func distance(a, b XY) float64 {
	defer reportPanic("distance")

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
var playerIDLock deadlock.Mutex

var objectTopID uint64
var objectIDLock deadlock.Mutex

func makePlayerID() uint32 {
	defer reportPanic("makePlayerID")

	playerIDLock.Lock()
	defer playerIDLock.Unlock()

	playerTopID++
	return playerTopID
}

func makeObjectID() uint64 {
	defer reportPanic("makObjectID")

	objectIDLock.Lock()
	defer objectIDLock.Unlock()

	objectTopID++
	return objectTopID
}
