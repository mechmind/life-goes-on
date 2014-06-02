package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

const (
	TOTAL_DAMSELS = 200
)

func main() {
	// set up logging
	f, err := os.Create("lgo.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	// panic protection
	defer logPanic()

	// seed random
	rand.Seed(time.Now().Unix())

	// create dispatcher
	dispatcher := NewDispatcher(singlePlayerRules)
	go dispatcher.Run()

	// create local render
	render := NewLocalRender()
	render.Init()

	// attach render (as player)
	dispatcher.AttachPlayer(render)

	// run render
	render.Run()
}
