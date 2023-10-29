package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type saveData struct {
	Version uint16
	Name    string
	ID      uint32
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

		err = os.WriteFile(fileName, outbuf.Bytes(), 0644)

		if err != nil {
			doLog(true, "WriteSector: WriteFile %v", err.Error())
			return
		}
	}
}

func loadWorld() {
	items, err := os.ReadDir(dataDir)
	if err != nil {
		doLog(true, "Unable to read data dir.")
		return
	}

	for _, item := range items {
		var sdat saveData

		if item.IsDir() {
			continue
		}
		fileName := item.Name()

		if !strings.HasSuffix(fileName, ".json") {
			continue
		}

		data, err := os.ReadFile(fileName)
		if err != nil {
			doLog(true, "Unable to read file: %v", fileName)
			continue
		}

		if data == nil {
			doLog(true, "File contains no data: %v", fileName)
			continue
		}

		buffer := bytes.NewBuffer(data)

		decoder := json.NewDecoder(buffer)

		err = decoder.Decode(&sdat)
		if err != nil {
			doLog(true, "Unable to decode json: %v", fileName)
			continue
		}

		if sdat.Version != areaVersion {
			doLog(true, "Incompatable area version: %v", fileName)
			continue
		}

		//Put data into an area
	}
}
