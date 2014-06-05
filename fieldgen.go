package main

func generateField(rules Rules) *Field {
	updates := make(chan *Field)
	field := NewField(FIELD_SIZE, FIELD_SIZE, updates)

	field.versus = rules.versus

	qp := NewQuarterPlan(CellCoord{32, 32})
	qp.CreateQuarters(field)

	return field
}

func findFreeCellInRange(field *Field, center CellCoord, r float32) CellCoord {
	for {
		cx := field.rng.Float32() * r * 2 - r + float32(center.X)
		cy := field.rng.Float32() * r * 2 - r + float32(center.Y)
		coord := CellCoord{int(cx), int(cy)}
		if field.CellAt(coord).Passable {
			return coord
		}
	}
}


func populateField(field *Field, rules Rules) {
	// zeds
	var swarm Agent = &ZedSwarm{}

	field.PlaceAgent(swarm)
	for i := 0; i < TOTAL_ZEDS + rules.moreZs; i++ {
		field.PlaceUnit(
			findFreeCellInRange(field, CellCoord{80, 80}, ZED_SPREAD_RADIUS).UnitCenter(),
			swarm, NewZed(field))
	}

	// damsels
	var crowd Agent = &DamselCrowd{}
	field.PlaceAgent(crowd)

	// respect additional Bs
	totalDamsels := TOTAL_DAMSELS + rules.moreBs
	for _, a := range field.Agents {
		if _, ok := a.(*Squad); ok {
			totalDamsels += rules.moreBsP
		}
	}

	for idx := 0; idx < totalDamsels; idx++ {
		var Coord UnitCoord
		for {
			Coord = UnitCoord{field.rng.Float32()*150 + 1, field.rng.Float32()*150 + 1}
			if field.CellAt(Coord.Cell()).Passable {
				break
			}
		}
		dam := NewDamsel(field)
		dam.WanderTarget = Coord
		field.PlaceUnit(dam.WanderTarget, crowd, dam)
	}
}

func placeSquad(field *Field, Id, Pid int, rules Rules) chan Order {
	var Orders = make(chan Order, SQUAD_ORDER_QUEUE_LEN)
	var squad Agent = &Squad{Orders: Orders, Pid: Pid, FireState: ORDER_FIRE, Versus: rules.versus}

	var sold1 = NewSoldier(field)
	var sold2 = NewSoldier(field)
	var sold3 = NewSoldier(field)
	var sold4 = NewSoldier(field)

	var cx, cy int
	if Id%2 == 1 {
		cx = 150
	}

	if Id/2 == 1 {
		cy = 150
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
