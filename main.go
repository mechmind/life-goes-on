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
	updates := make(chan *Field)
	field := NewField(1024, 1024, updates)
	time := NewTime(10, field)

	// make obstacles
	// generate quarters
	qp := NewQuarterPlan(CellCoord{32, 32})
	qp.CreateQuarters(field)

	//*
	var orders = make(chan Order, SQUAD_ORDER_QUEUE_LEN)
	var mainSquad Agent = &Squad{orders: orders, fireState: ORDER_FIRE}
	var sold1 = NewSoldier(field)
	//*
	var sold2 = NewSoldier(field)
	var sold3 = NewSoldier(field)
	var sold4 = NewSoldier(field)
	//*/

	field.PlaceAgent(mainSquad)
	field.PlaceUnit(CellCoord{3, 1}.UnitCenter(), mainSquad, sold1)
	//*
	field.PlaceUnit(CellCoord{1, 1}.UnitCenter(), mainSquad, sold2)
	field.PlaceUnit(CellCoord{1, 3}.UnitCenter(), mainSquad, sold3)
	field.PlaceUnit(CellCoord{3, 3}.UnitCenter(), mainSquad, sold4)
	//*/

	//*
	var swarm Agent = &ZedSwarm{}
	var zed1 = NewZed(field)
	var zed2 = NewZed(field)

	field.PlaceAgent(swarm)
	field.PlaceUnit(UnitCoord{80, 85}, swarm, zed1)
	field.PlaceUnit(UnitCoord{85, 80}, swarm, zed2)

	//*/

	var crowd Agent = &DamselCrowd{}
	field.PlaceAgent(crowd)

	/*
	for idx := 0; idx < TOTAL_DAMSELS; idx++ {
		var coord UnitCoord
		for {
			coord = UnitCoord{rand.Float32()*100 + 1, rand.Float32()*100 + 1}
			if field.CellAt(coord.Cell()).passable {
				break
			}
		}
		dam := NewDamsel(field)
		dam.wanderTarget = coord
		field.PlaceUnit(dam.wanderTarget, crowd, dam)
	}
	//*/

	// FIXME(pathfind): debugging
	// make tight corridor
	/*
	for i := 2; i < 10; i++ {
		field.CellAt(CellCoord{i, 2}).passable = false
		field.CellAt(CellCoord{2, i}).passable = false
	}
	//*/
	//dam := NewDamsel(field)
	//dam.wanderTarget = UnitCoord{5.5, 15.5}
	//field.PlaceUnit(UnitCoord{4.97, 14.95}, crowd, dam)

	go time.Run()
	RunTUI(updates, orders)
}
