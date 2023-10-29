package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type saveData struct {
	Version uint16
	Objects []*worldObject
}

const areaVersion = 1
const dataDir = "data"

func saveWorld() {
	var sdat saveData

	for _, area := range areaList {
		for _, chunk := range area.Chunks {
			sdat.Objects = append(sdat.Objects, chunk.WorldObjects...)
		}

		area.arealock.Lock()
		area.Version = areaVersion

		outbuf := new(bytes.Buffer)
		enc := json.NewEncoder(outbuf)
		enc.SetIndent("", "\t")

		fileName := fmt.Sprintf("%v/%v.json", dataDir, area.Name)

		err := enc.Encode(sdat)
		if err != nil {
			doLog(true, "WriteSector: enc.Encode %v", err.Error())
			area.arealock.Unlock()
			return
		}
		area.dirty = false
		area.arealock.Unlock()

		os.MkdirAll(dataDir, 0755)
		_, err = os.Create(fileName)

		if err != nil {
			doLog(true, "WriteSector: os.Create %v", err.Error())
			return
		}

		err = ioutil.WriteFile(fileName, outbuf.Bytes(), 0644)

		if err != nil {
			doLog(true, "WriteSector: WriteFile %v", err.Error())
			return
		}
	}
}
