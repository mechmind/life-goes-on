package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	// set up logging
	f, err := os.Create("lgo.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	// seed random
	rand.Seed(time.Now().Unix())
	updates := make(chan *Field)
	field := NewField(1024, 1024, updates)
	time := NewTime(10, field)

	var mainSquad Agent = &Squad{}
	var sold1 = NewSoldier(field)
	var sold2 = NewSoldier(field)
	var sold3 = NewSoldier(field)
	var sold4 = NewSoldier(field)

	field.PlaceAgent(mainSquad)
	field.PlaceUnit(UnitCoord{1, 1}, mainSquad, sold1)
	field.PlaceUnit(UnitCoord{3, 1}, mainSquad, sold2)
	field.PlaceUnit(UnitCoord{1, 3}, mainSquad, sold3)
	field.PlaceUnit(UnitCoord{3, 3}, mainSquad, sold4)

	var swarm Agent = &ZedSwarm{}
	var zed1 = NewZed(field)
	var zed2 = NewZed(field)

	field.PlaceAgent(swarm)
	field.PlaceUnit(UnitCoord{80, 85}, swarm, zed1)
	field.PlaceUnit(UnitCoord{85, 80}, swarm, zed2)

	var crowd Agent = &DamselCrowd{}
	field.PlaceAgent(crowd)

	for idx := 0; idx < 135; idx++ {
		coord := UnitCoord{rand.Float32() * 100, rand.Float32() * 100}
		dam := NewDamsel(field)
		dam.wanderTarget = coord
		field.PlaceUnit(dam.wanderTarget, crowd, dam)
	}

	go time.Run()
	RunTUI(updates)
}
