package main

import (
	"math/rand"
)

func main() {
	updates := make(chan *Field)
	field := NewField(1024, 1024, updates)
	time := NewTime(10, field)

	var mainSquad Agent = &Squad{}
	var sold1 = &Soldier{Walker: Walker{0.65, 0.25, 0.60}, Chaser: Chaser{-1},
		Gunner: Gunner{fireRange: 5, gunDamage: 5}, health: 100, field: field}
	var sold2 = new(Soldier)
	*sold2 = *sold1 // copy soldier

	field.PlaceAgent(mainSquad)
	field.PlaceUnit(UnitCoord{1, 1}, mainSquad, sold1)
	field.PlaceUnit(UnitCoord{2, 1}, mainSquad, sold2)

	var swarm Agent = &ZedSwarm{}
	var zed1 = NewZed(field)
	var zed2 = NewZed(field)

	field.PlaceAgent(swarm)
	field.PlaceUnit(UnitCoord{15, 15}, swarm, zed1)
	field.PlaceUnit(UnitCoord{15, 17}, swarm, zed2)

	var crowd Agent = &DamselCrowd{}
	field.PlaceAgent(crowd)

	for idx := 0; idx < 35; idx++ {
		coord := UnitCoord{rand.Float32() * 100, rand.Float32() * 100}
		dam := &Damsel{Walker: Walker{0.30, 0.10, 0.25}, wanderTarget: coord, lastAttacker: -1,
			health: 75, field: field}
		field.PlaceUnit(dam.wanderTarget, crowd, dam)
	}

	go time.Run()
	RunTUI(updates)
}
