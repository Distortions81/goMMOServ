package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

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

func uint64ToByteArray(i uint64) []byte {
	defer reportPanic("uint64ToByteArray")

	byteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteArray, i)
	return byteArray
}

func uint32ToByteArray(i uint32) []byte {
	defer reportPanic("uint32ToByteArray")

	byteArray := make([]byte, 4)
	binary.LittleEndian.PutUint32(byteArray, i)
	return byteArray
}

func uint16ToByteArray(i uint16) []byte {
	defer reportPanic("uint16ToByteArray")
	byteArray := make([]byte, 2)
	binary.LittleEndian.PutUint16(byteArray, i)
	return byteArray
}

func uint8ToByteArray(i uint8) []byte {
	defer reportPanic("uint8ToByteArray")

	byteArray := make([]byte, 1)
	byteArray[0] = byte(i)
	return byteArray
}

func byteArrayToUint8(i []byte) uint8 {
	defer reportPanic("byteArrayToUint8")

	if len(i) < 1 {
		return 0
	}
	return uint8(i[0])
}

func byteArrayToUint16(i []byte) uint16 {
	defer reportPanic("byteArrayToUint16")

	if len(i) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(i)
}

func byteArrayToUint32(i []byte) uint32 {
	defer reportPanic("byteArrayToUint32")

	if len(i) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(i)
}

func byteArrayToUint64(i []byte) uint64 {
	defer reportPanic("byteArrayToUint64")

	if len(i) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(i)
}

func xyToByteArray(pos XY) []byte {
	byteArray := make([]byte, 16)
	binary.LittleEndian.PutUint32(byteArray[0:7], pos.X)
	binary.LittleEndian.PutUint32(byteArray[8:16], pos.Y)
	return byteArray
}
func byteArrayToXY(pos *XY, i []byte) bool {

	if len(i) < 16 {
		doLog(true, "byteArrayToXY: data invalid")
		return true
	}

	pos.X = binary.LittleEndian.Uint32(i[0:7])
	pos.Y = binary.LittleEndian.Uint32(i[8:16])
	return false
}

func convPos(pos XY) XYs {
	return XYs{X: int32(pos.X - xyHalf), Y: int32(pos.Y - xyHalf)}
}
