package main

func main() {
	field := NewField(1024, 1024)
	time := NewTime(10, field)

	var mainSquad Agent = &Squad{}
	var sold1 = &Soldier{Walker: Walker{0.45, 0.065, 0.11}, field: field}
	var sold2 = &Soldier{Walker: Walker{0.20, 0.02,  0.20}, field: field}

	field.PlaceAgent(mainSquad)
	field.PlaceUnit(UnitCoord{1, 1}, mainSquad, sold1)
	field.PlaceUnit(UnitCoord{2, 1}, mainSquad, sold2)
	time.Run()
}
