package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

const (
	hdFileName = "heapDump.dat"
	pLogName   = "panic.log"
)

func reportPanic(format string, args ...interface{}) {
	if r := recover(); r != nil {

		doLog(false, "Writing '%v' file.", hdFileName)
		f, err := os.Create(hdFileName)
		if err == nil {
			debug.WriteHeapDump(f.Fd())
			f.Close()
			doLog(true, "wrote heapDump")
		} else {
			doLog(false, "Failed to write '%v' file.", hdFileName)
		}

		_, filename, line, _ := runtime.Caller(4)
		input := fmt.Sprintf(format, args...)
		buf := fmt.Sprintf(
			"(GAME CRASH)\nBUILD:v%v-%v\nLabel:%v File: %v Line: %v\nError:%v\n\nStack Trace:\n%v\n",
			version, buildInfo, input, filepath.Base(filename), line, r, string(debug.Stack()))

		os.WriteFile(pLogName, []byte(buf), 0660)
		doLog(true, "wrote %v", pLogName)

	}
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
