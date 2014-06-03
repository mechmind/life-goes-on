package main

func generateField() *Field {
	updates := make(chan *Field)
	field := NewField(FIELD_SIZE, FIELD_SIZE, updates)

	qp := NewQuarterPlan(CellCoord{32, 32})
	qp.CreateQuarters(field)

	return field
}

func populateField(field *Field) {
	// zeds
	var swarm Agent = &ZedSwarm{}
	var zed1 = NewZed(field)
	var zed2 = NewZed(field)

	field.PlaceAgent(swarm)
	field.PlaceUnit(UnitCoord{80, 85}, swarm, zed1)
	field.PlaceUnit(UnitCoord{85, 80}, swarm, zed2)

	// damsels
	var crowd Agent = &DamselCrowd{}
	field.PlaceAgent(crowd)

	for idx := 0; idx < TOTAL_DAMSELS; idx++ {
		var coord UnitCoord
		for {
			coord = UnitCoord{field.rng.Float32()*100 + 1, field.rng.Float32()*100 + 1}
			if field.CellAt(coord.Cell()).passable {
				break
			}
		}
		dam := NewDamsel(field)
		dam.WanderTarget = coord
		field.PlaceUnit(dam.WanderTarget, crowd, dam)
	}
}

func placeSquad(field *Field, Id, Pid int) chan Order {
	var Orders = make(chan Order, SQUAD_ORDER_QUEUE_LEN)
	var squad Agent = &Squad{Orders: Orders, Pid: Pid, FireState: ORDER_FIRE}

	var sold1 = NewSoldier(field)
	var sold2 = NewSoldier(field)
	var sold3 = NewSoldier(field)
	var sold4 = NewSoldier(field)

	var cx, cy int
	if Id%2 == 1 {
		cx = 100
	}

	if Id/2 == 1 {
		cy = 100
	}

	field.PlaceAgent(squad)

	field.PlaceUnit(findFreeCellNearby(field, CellCoord{1, 1}.Add(cx, cy)).UnitCenter(),
		squad, sold1)
	field.PlaceUnit(findFreeCellNearby(field, CellCoord{3, 1}.Add(cx, cy)).UnitCenter(),
		squad, sold2)
	field.PlaceUnit(findFreeCellNearby(field, CellCoord{1, 3}.Add(cx, cy)).UnitCenter(),
		squad, sold3)
	field.PlaceUnit(findFreeCellNearby(field, CellCoord{3, 3}.Add(cx, cy)).UnitCenter(),
		squad, sold4)

	return Orders
}

func findFreeCellNearby(field *Field, desiredCell CellCoord) CellCoord {
	return desiredCell
}
