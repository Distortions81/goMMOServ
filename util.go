package main

import (
	"bytes"
	"compress/zlib"
	"io"
	"math"
	"sync"
)

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

func convPos(pos XY) XYs {
	return XYs{X: int32(pos.X - xyHalf), Y: int32(pos.Y - xyHalf)}
}

var playerTopID uint32
var playerIDLock sync.Mutex

func makePlayerID() uint32 {
	playerIDLock.Lock()
	defer playerIDLock.Unlock()

	playerTopID++
	return playerTopID
}
