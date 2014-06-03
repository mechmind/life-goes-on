package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"
)

const (
	TOTAL_DAMSELS = 200
)

var listen = flag.String("listen", "", "start server on given address")
var connect = flag.String("connect", "", "connect to server on giving address")
var logfile = flag.String("log", "lgo.log", "log to that file")

func main() {
	flag.Parse()
	// set up logging
	f, err := os.Create(*logfile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	// panic protection
	defer logPanic()

	// seed random
	rand.Seed(time.Now().Unix())

	var attachTo interface {
		AttachPlayer(Render)
	}
	if *connect != "" {
		// connect to remote game
		remote, err := ConnectRemoteGame(*connect)
		if err != nil {
			log.Fatal(err)
		}

		go remote.Run()
		attachTo = remote
	} else {
		// start local game
		// create dispatcher
		rules := singlePlayerRules
		if *listen != "" {
			rules = duelRules
		}

		dispatcher := NewDispatcher(rules)
		go dispatcher.Run()
		attachTo = dispatcher

		if *listen != "" {
			server, err := CreateServer(dispatcher, *listen)
			if err != nil {
				log.Fatal(err)
			}
			go server.Serve()
		}
	}

	// create local render
	render := NewLocalRender()
	render.Init()

	// attach render (as player)
	attachTo.AttachPlayer(render)

	// run render
	render.Run()
}
