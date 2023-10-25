package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

const (
	hdFileName = "heapDump.dat"
	pLogName   = "panic.log"
	version    = "0.0.1"
	buildInfo  = "dev"
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
		send_chat("Warning: Server paniced, but recovered (wrote to log)")
	}
}
