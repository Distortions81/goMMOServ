package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

/* Generic unzip []byte */
func UncompressZip(data []byte) []byte {
	b := bytes.NewReader(data)

	z, _ := zlib.NewReader(b)
	defer z.Close()

	p, err := io.ReadAll(z)
	if err != nil {
		return nil
	}
	return p
}

/* Generic zip []byte */
func CompressZip(data []byte) []byte {
	var b bytes.Buffer
	w, _ := zlib.NewWriterLevel(&b, zlib.BestCompression)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

func uint64ToByteArray(i uint64) []byte {
	byteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteArray, i)
	return byteArray
}

func uint32ToByteArray(i uint32) []byte {
	byteArray := make([]byte, 4)
	binary.LittleEndian.PutUint32(byteArray, i)
	return byteArray
}

func uint16ToByteArray(i uint16) []byte {
	byteArray := make([]byte, 2)
	binary.LittleEndian.PutUint16(byteArray, i)
	return byteArray
}

func uint8ToByteArray(i uint8) []byte {
	byteArray := make([]byte, 1)
	byteArray[0] = byte(i)
	return byteArray
}

func byteArrayToUint8(i []byte) uint8 {
	if len(i) < 1 {
		return 0
	}
	return uint8(i[0])
}

func byteArrayToUint16(i []byte) uint16 {
	if len(i) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(i)
}

func byteArrayToUint32(i []byte) uint32 {
	if len(i) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(i)
}

func byteArrayToUint64(i []byte) uint64 {
	if len(i) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(i)
}
