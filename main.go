package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime/pprof"
	"time"
)

var (
	fileServer http.Handler
	areaList   []*areaData
	gTestMode  bool
)

func main() {
	defer reportPanic("main")
	defer time.Sleep(time.Second)

	//Parse launch params
	devMode := flag.Bool("dev", false, "dev mode enable")
	bindIP := flag.String("ip", "", "IP to bind to")
	bindPort := flag.Int("port", 443, "port to bind to for HTTPS")
	testMode := flag.Bool("test", false, "load many test characters")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		doLog(true, "pprof started")
		pprof.StartCPUProfile(f)
		go func() {
			time.Sleep(time.Minute)
			pprof.StopCPUProfile()
			doLog(true, "pprof complete")
		}()
	}

	gTestMode = *testMode

	//Start logger
	startLog()
	logDaemon()

	/* make test area */
	tmp := &areaData{Name: "test", ID: 0, Chunks: make(map[XY]*chunkData)}
	areaList = append(areaList, tmp)
	loadWorld()

	go autoSaveWorld()

	processLock.Lock()

	//Test zombies!
	for x := 0; x < 100; x++ {

		startPos := XYf32{X: 0, Y: 0}

		newCreature := &playerData{
			area:         areaList[0],
			creatureData: &creatureData{id: IID{Section: 1, Num: 0, UID: makeCreatureID()}, mode: CRE_ATTACK},
			pos:          startPos, health: 100,
			dir: DIR_S, moveDir: DIR_NONE, VALID: true, mode: PMODE_ATTACK}

		addPlayerToWorld(areaList[0], startPos, newCreature)

		for !movePlayer(newCreature, true) {
			randx := 10000 - (rand.Float32() * 20000.0)
			randy := 10000 - (rand.Float32() * 20000.0)
			startPos = XYf32{X: randx, Y: randy}
			movePlayerChunk(newCreature.area, startPos, newCreature)
			fmt.Println("Spawn blocked.")
		}
	}

	processLock.Unlock()

	processGame()

	/* Download server start */
	fileServer = http.FileServer(http.Dir("www"))

	/* Create HTTPS server */
	server := &http.Server{}
	server.Addr = fmt.Sprintf("%v:%v", *bindIP, *bindPort)

	/* HTTPS server */
	http.HandleFunc("/gs", gsHandler) //websocket
	http.HandleFunc("/", siteHandler) //wasm download site

	//If not in development mode, setup an origin check
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if !*devMode && origin != "https://gommo.go-game.net" {
			doLog(true, "Connection failed origin check: %v", r.RemoteAddr)
			return false
		}
		return true
	}

	/* Start server*/
	doLog(true, "Starting server...")

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("%v:80", *bindIP), http.HandlerFunc(redirectToTls)); err != nil {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 5)

			filePath := "fullchain.pem"
			initialStat, erra := os.Stat(filePath)

			if erra != nil {
				continue
			}

			for initialStat != nil {
				time.Sleep(time.Second * 5)

				stat, errb := os.Stat(filePath)
				if errb != nil {
					break
				}

				if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
					doLog(true, "Cert updated, closing.")
					time.Sleep(time.Second * 5)
					os.Exit(0)
					break
				}
			}

		}
	}()

	err := server.ListenAndServeTLS("fullchain.pem", "privkey.pem")
	if err != nil {
		doLog(true, "ListenAndServeTLS: %v", err)
		return
	}

	doLog(true, "Goodbye.")
}
