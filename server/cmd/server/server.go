package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/giongto35/gowog/server/game"
	"github.com/giongto35/gowog/server/game/ws"
	"github.com/gorilla/websocket"
	"github.com/pkg/profile"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")
var cpuprofile = flag.Bool("cpuprofile", false, "Enable CPUProfile")
var memprofile = flag.Bool("memprofile", false, "Enable MemProfile")
var disablelog = flag.Bool("disablelog", false, "Disable log")
var clientBuild = flag.String("prod", "", "is production")

var upgrader = websocket.Upgrader{} // use default options
var hub = ws.NewHub()
var gameMaster = game.NewGame(hub)

var ErrDuplicatedAddress = errors.New("Duplicated Address")
var exist = map[string]ws.Client{}

// serveWs handles websocket requests from the peer.
func connect(w http.ResponseWriter, r *http.Request) {
	// Upgrade request response to socket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Get remote address
	var remoteAddr string
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		remoteAddr = parts[0]
	}

	fmt.Println("Registering ", remoteAddr)
	// If exist, we have duplication connection -> end
	// TODO: invalidate exist when client disconnect
	if _, ok := exist[remoteAddr]; ok {
		// TODO: Send duplicate message error
		return
	}
	clientID := ws.NewClient(conn, hub, w, r)
	exist[remoteAddr] = clientID

	// We need to register client from hub.
	<-hub.Register(clientID)
	gameMaster.NewPlayerConnect(clientID)
}

func main() {
	// Running on one core only
	runtime.GOMAXPROCS(2)
	flag.Parse()
	if *disablelog {
		log.SetOutput(ioutil.Discard)
	}
	// CPU profile
	if *cpuprofile {
		fmt.Println("Profiling CPU")
		defer profile.Start().Stop()
	}
	// Memory profile
	if *memprofile {
		fmt.Println("Profiling MemProfile")
		defer profile.Start(profile.MemProfile).Stop()
	}

	// If there is clientBuild flag, we return the client build for index
	if *clientBuild != "" {
		fmt.Println("loading file from ", *clientBuild)
		http.Handle("/", http.FileServer(http.Dir(*clientBuild)))

	}

	// HTTP setup
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	log.SetFlags(0)
	// Websocket endpoint
	http.HandleFunc("/game/", connect)

	fmt.Println("Listening to ", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
	fmt.Println("Stop Listening to ", addr)
}
