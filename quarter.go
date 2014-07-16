package main

const (
	QUARTER_BLOCK_SIZE           = 6
	QUARTER_HOUSE_SIZE           = 32
	QUARTER_PADDING              = 2
	QUARTER_PROBABILITY          = 35
	QUARTER_INTWALL_PROBABILITY  = 75
	QUARTER_BACKDOOR_PROBABILITY = 33
)

const (
	QUARTER_HORIZONTAL = iota
	QUARTER_VERTICAL
)

type QuarterPlan struct {
	size CellCoord
}

func NewQuarterPlan(size CellCoord) *QuarterPlan {
	qp := &QuarterPlan{size}
	return qp
}

func (qp *QuarterPlan) CreateQuarters(f *Field) {
	for i := 0; i < qp.size.X; i++ {
		for j := 0; j < qp.size.Y; j++ {
			// roll dices for house in that block
			if f.rng.Int31n(100) < QUARTER_PROBABILITY {
				qp.MakeHouse(f, CellCoord{i, j})
			} else {
				// FIXME: debug
				// make a bush line
				xLow := i*QUARTER_HOUSE_SIZE + QUARTER_PADDING
				yLow := j*QUARTER_HOUSE_SIZE + QUARTER_PADDING
				for xx := xLow; xx < xLow + 15; xx++ {
					f.CellAt(CellCoord{xx, yLow}).Object = referenceObjects[OBJECT_BUSH]
				}
				// make a barricade
				for yy := yLow+1; yy < yLow + 15; yy++ {
					f.CellAt(CellCoord{xLow, yy}).Object = referenceObjects[OBJECT_BARRICADE]
				}
			}
		}
	}
}

func (qp *QuarterPlan) MakeHouse(f *Field, Coord CellCoord) {
	// select size for new house (2..4) x (2..4)
	size := CellCoord{f.rng.Intn(3) + 2, f.rng.Intn(3) + 2}
	// select placement for topleft corner
	var corner CellCoord
	if size.X != 4 {
		corner.X = f.rng.Intn(4 - size.X)
	}
	if size.Y != 4 {
		corner.Y = f.rng.Intn(4 - size.Y)
	}

	// merge foundament into field
	xLow := Coord.X*QUARTER_HOUSE_SIZE + corner.X*QUARTER_BLOCK_SIZE + QUARTER_PADDING
	yLow := Coord.Y*QUARTER_HOUSE_SIZE + corner.Y*QUARTER_BLOCK_SIZE + QUARTER_PADDING

	xHig := Coord.X*QUARTER_HOUSE_SIZE + (corner.X+size.X)*QUARTER_BLOCK_SIZE + QUARTER_PADDING
	yHig := Coord.Y*QUARTER_HOUSE_SIZE + (corner.Y+size.Y)*QUARTER_BLOCK_SIZE + QUARTER_PADDING

	for i := xLow; i <= xHig; i++ {
		f.CellAt(CellCoord{i, yLow}).Object = referenceObjects[OBJECT_WALL]
		f.CellAt(CellCoord{i, yHig}).Object = referenceObjects[OBJECT_WALL]
	}
	for j := yLow; j <= yHig; j++ {
		f.CellAt(CellCoord{xLow, j}).Object = referenceObjects[OBJECT_WALL]
		f.CellAt(CellCoord{xHig, j}).Object = referenceObjects[OBJECT_WALL]
	}
	// make main door
	hasBackDoor := f.rng.Intn(100) < QUARTER_BACKDOOR_PROBABILITY
	if f.rng.Int31n(1) == 0 {
		// X-axis door
		doorPos := f.rng.Intn((xHig-xLow-1)/2)*2 + xLow + 1
		bdoorPos := f.rng.Intn((xHig-xLow-1)/2)*2 + xLow + 1
		if f.rng.Int31n(1) == 0 {
			// bottom wall
			f.CellAt(CellCoord{doorPos, yLow}).Object = referenceObjects[OBJECT_EMPTY]
			if hasBackDoor {
				f.CellAt(CellCoord{bdoorPos, yHig}).Object = referenceObjects[OBJECT_EMPTY]
			}
		} else {
			// top wall
			f.CellAt(CellCoord{doorPos, yHig}).Object = referenceObjects[OBJECT_EMPTY]
			if hasBackDoor {
				f.CellAt(CellCoord{bdoorPos, yLow}).Object = referenceObjects[OBJECT_EMPTY]
			}
		}
	} else {
		// Y-axis door
		doorPos := f.rng.Intn((yHig-xLow-1)/2)*2 + yLow + 1
		bdoorPos := f.rng.Intn((yHig-xLow-1)/2)*2 + yLow + 1
		if f.rng.Int31n(1) == 0 {
			// left wall
			f.CellAt(CellCoord{xLow, doorPos}).Object = referenceObjects[OBJECT_EMPTY]
			if hasBackDoor {
				f.CellAt(CellCoord{xHig, bdoorPos}).Object = referenceObjects[OBJECT_EMPTY]
			}
		} else {
			// right wall
			f.CellAt(CellCoord{xHig, doorPos}).Object = referenceObjects[OBJECT_EMPTY]
			if hasBackDoor {
				f.CellAt(CellCoord{xLow, bdoorPos}).Object = referenceObjects[OBJECT_EMPTY]
			}
		}
	}

	// make internal wall if house is big
	if size.X > 2 {
		wallX := f.rng.Intn(QUARTER_BLOCK_SIZE/2*(size.X-2))*2 + xLow + QUARTER_BLOCK_SIZE

		var j int
		for j = yLow + 1; j <= yHig; j++ {
			cell := f.CellAt(CellCoord{wallX, j})
			if cell.Passable {
				cell.Object = referenceObjects[OBJECT_WALL]
			} else {
				break
			}
		}
		// make a door in that wall
		doorPos := f.rng.Intn((j-yLow)/2)*2 + yLow + 1
		f.CellAt(CellCoord{wallX, doorPos}).Object = referenceObjects[OBJECT_EMPTY]
	}

	if size.Y > 2 && f.rng.Intn(100) < QUARTER_INTWALL_PROBABILITY {
		wallY := f.rng.Intn(QUARTER_BLOCK_SIZE/2*(size.Y-2))*2 + yLow + QUARTER_BLOCK_SIZE

		var i int
		for i = xLow + 1; i <= xHig; i++ {
			cell := f.CellAt(CellCoord{i, wallY})
			if cell.Passable {
				cell.Object = referenceObjects[OBJECT_WALL]
			} else {
				break
			}
		}
		// make a door in that wall
		doorPos := f.rng.Intn((i-xLow)/2)*2 + xLow + 1
		f.CellAt(CellCoord{doorPos, wallY}).Object = referenceObjects[OBJECT_EMPTY]
	}

}
