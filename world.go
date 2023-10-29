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
	ID      uint16
	Objects []*worldObject
}

const areaVersion = 1
const dataDir = "data"
const areaDir = "areas"
const suffix = ".json"

func saveWorld() {
	var sdat saveData

	for a, area := range areaList {
		for _, chunk := range area.Chunks {
			sdat.Objects = append(sdat.Objects, chunk.WorldObjects...)
		}

		areaList[a].arealock.Lock()
		sdat.Version = areaVersion

		outbuf := new(bytes.Buffer)
		enc := json.NewEncoder(outbuf)
		enc.SetIndent("", "\t")

		filePath := fmt.Sprintf("%v/%v/%v%v", dataDir, areaDir, area.Name, suffix)

		err := enc.Encode(sdat)
		if err != nil {
			doLog(true, "WriteSector: enc.Encode %v", err.Error())
			areaList[a].arealock.Unlock()
			return
		}
		areaList[a].dirty = false
		areaList[a].arealock.Unlock()

		os.MkdirAll(dataDir+"/"+areaDir, 0755)
		_, err = os.Create(filePath)

		if err != nil {
			doLog(true, "WriteSector: os.Create %v", err.Error())
			return
		}

		err = os.WriteFile(filePath, outbuf.Bytes(), 0644)

		if err != nil {
			doLog(true, "WriteSector: WriteFile %v", err.Error())
			return
		}
	}
}

func loadWorld() {
	doLog(true, "Loading areas...")
	items, err := os.ReadDir(dataDir + "/" + areaDir)
	if err != nil {
		doLog(true, "Unable to read data dir.")
		return
	}

	fileFound := 0
	for _, item := range items {
		var sdat saveData

		if item.IsDir() {
			continue
		}
		fileFound++

		fileName := item.Name()
		areaName := strings.TrimSuffix(fileName, suffix)

		if !strings.HasSuffix(fileName, suffix) {
			continue
		}

		data, err := os.ReadFile(dataDir + "/" + areaDir + "/" + fileName)
		if err != nil {
			doLog(true, "Unable to read file: %v", fileName)
			continue
		} else {
			doLog(true, "Reading %v", fileName)
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
		newArea := &areaData{Version: areaVersion, Name: areaName, ID: sdat.ID}
		newArea.Chunks = make(map[XY]*chunkData)

		numObj := 0
		for _, obj := range sdat.Objects {
			addWorldObject(newArea, obj.Pos, obj)
			numObj++
		}

		doLog(true, "Loaded %v objects, %v files in dir.", numObj, fileFound)

		areaList[newArea.ID] = newArea
	}
}
