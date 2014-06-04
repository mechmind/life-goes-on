package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

const (
	TOTAL_DAMSELS = 300
)

var listen = flag.String("listen", "", "start server on given address")
var connect = flag.String("connect", "", "connect to server on giving address")
var logfile = flag.String("log", "lgo.log", "log to that file")
var standalone = flag.Bool("standalone", false, "run server as standalone")

var defaultServerAddr string

func main() {
	flag.Parse()
	if defaultServerAddr != "" && *connect == "" {
		*connect = defaultServerAddr
	}

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
		AttachPlayer(Render) int
	}

	if *connect != "" {
		// connect to remote game
		log.Printf("main: connecting to remote game at '%s'", *connect)
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
			rules = coopRules
			//rules = singlePlayerRules
		}

		log.Println("main: starting dispatcher")
		dispatcher := NewDispatcher(rules)
		go dispatcher.Run()
		attachTo = dispatcher

		if *listen != "" {
			log.Printf("main: starting server at '%s'", *listen)
			server, err := CreateServer(dispatcher, *listen)
			if err != nil {
				log.Fatal(err)
			}
			go server.Serve()
		}
	}

	if *listen != "" && *standalone {
		fmt.Println("server started on", *listen)
		for {
			time.Sleep(time.Hour)
		}
	} else {
		// create local render
		log.Println("main: creating local render")
		render := NewLocalRender()
		render.Init()

		log.Println("main: attaching to game")
		// attach render (as player)
		attachTo.AttachPlayer(render)

		log.Println("main: ready to play!")
		// run render
		render.Run()
	}
}
