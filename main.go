package main

func main() {
	field := NewField(1024, 1024)
	time := NewTime(10, field)

	var mainSquad Agent = &Squad{}
	var sold1 = &Soldier{Walker: Walker{0.45, 0.25, 0.60}, Chaser: Chaser{-1},
		Gunner: Gunner{fireRange: 10, gunDamage: 15}, health: 100, field: field}
	var sold2 = new(Soldier)
	*sold2 = *sold1 // copy soldier

	field.PlaceAgent(mainSquad)
	field.PlaceUnit(UnitCoord{1, 1}, mainSquad, sold1)
	field.PlaceUnit(UnitCoord{2, 1}, mainSquad, sold2)

	var swarm Agent = &ZedSwarm{}
	var zed1 = &Zed{Walker: Walker{0.20, 0.02, 0.20}, health: 140, field: field}

	field.PlaceAgent(swarm)
	field.PlaceUnit(UnitCoord{15, 15}, swarm, zed1)

	time.Run()
}
