package main

const (
	FLOAT_ERROR           = 0.000001
	FIELD_BACKBUFFER_SIZE = 3
)

const (
	SLOPE_UP, SLOPE_UP_SHIFT = 1 << iota, iota
	SLOPE_RIGHT, SLOPE_RIGHT_SHIFT
	SLOPE_DOWN, SLOPE_DOWN_SHIFT
	SLOPE_LEFT, SLOPE_LEFT_SHIFT
)

const (
	PS_PASSABLE = Passability(iota)
	PS_IMPASSABLE
)

var nopAgent NopAgent

var fieldBackbuffer = make(chan *Field, FIELD_BACKBUFFER_SIZE)

type Passability int

type Field struct {
	xSize, ySize int
	cells        []Cell
	agents       []Agent
	units        []UnitPresence
	updates      chan *Field

	// FIXME(pathfind): remove after debugging
	pathfinder *PathFinder
}

func NewField(xSize, ySize int, updates chan *Field) *Field {
	field := &Field{xSize, ySize, make([]Cell, xSize*ySize), nil, nil, updates, nil}
	field.makePassableField()
	field.computeSlopes()
	return field
}

func copyField(f *Field) *Field {
	var bb *Field
	select {
	case bb = <-fieldBackbuffer:
	default:
		bb = &Field{f.xSize, f.ySize, make([]Cell, f.xSize*f.ySize), nil, nil, nil, nil}
	}

	copy(bb.cells, f.cells)
	bb.units = append(bb.units[:0], f.units...)
	bb.agents = append(bb.agents[:0], f.agents...)
	bb.pathfinder = f.pathfinder // FIXME(pathfind)
	bb.updates = fieldBackbuffer

	return bb
}

func (f *Field) Tick(tick int64) {
	view := &FieldView{f}

	for _, agent := range f.agents {
		if thinker, ok := agent.(Thinker); ok {
			thinker.Think(view, tick)
		}
	}

	for _, up := range f.units {
		up.agent.HandleUnit(view, up.unit, up.coord)
	}

	select {
	case f.updates <- copyField(f):
	default:
	}
}

func (f *Field) PlaceUnit(c UnitCoord, agent Agent, u Unit) error {
	f.units = append(f.units, UnitPresence{c, agent, u})
	u.SetID(len(f.units) - 1)
	agent.AttachUnit(u)
	return nil
}

func (f *Field) PlaceAgent(a Agent) error {
	f.agents = append(f.agents, a)
	return nil
}

func (f *Field) ReplaceUnit(id int, c UnitCoord, agent Agent, u Unit) error {
	f.units[id] = UnitPresence{c, agent, u}
	u.SetID(id)
	agent.AttachUnit(u)
	return nil
}

// unit/agent api
func (f *Field) CellAt(c CellCoord) *Cell {
	return &f.cells[c.Y*f.xSize+c.X]
}

func (f *Field) UnitByID(id int) (UnitCoord, Unit) {
	return f.units[id].coord, f.units[id].unit
}

func (f *Field) UnitsInRange(center UnitCoord, radius float32) []UnitPresence {
	var units []UnitPresence

	for _, up := range f.units {
		if center.Distance(up.coord) < radius {
			units = append(units, up)
		}
	}
	return units
}

// return true if have line of sight from 'from' to 'to'
func (f *Field) HaveLOS(from, to UnitCoord) bool {
	toward := NormTowardCoord(from, to)

	current := from
	for {
		// always check next and current cell passability because we can advance 2 cells
		// on one step
		if f.CellAt(current.Cell()).passable == false {
			return false
		}

		if current == to {
			return true
		}

		nextCell := from.Cell().AddCoord(NextCellCoord(from, toward)).Bound(0, 0, 1024, 1024)
		if f.CellAt(nextCell).passable == false {
			return false
		}

		if current.Distance(to) < 1 {
			current = to
		} else {
			current = current.AddCoord(toward)
		}
	}
}

func (f *Field) CheckPassability(src, dst CellCoord) Passability {
	dstCell := f.CellAt(dst)
	srcCell := f.CellAt(src)
	if !dstCell.passable {
		//log.Println("field: cannot pass to dst", dst, "- is a wall")
		return PS_IMPASSABLE
	}
	if iabs(int(srcCell.elevation-dstCell.elevation)) > 2 {
		//log.Println("field: cannot pass to dst", dst, "- elevation is too high")
		return PS_IMPASSABLE
	}
	if src.Distance(dst) > 1 {
		// diagonal move
		direction := dst.AddCoord(src.Mult(-1))
		s1 := src.AddCoord(direction.ClockwiseSibling())
		s2 := src.AddCoord(direction.CounterclockwiseSibling())
		//log.Println("field: d:", direction, "siblings:", s1, s2)
		if !f.CellAt(s1).passable || !f.CellAt(s2).passable {
			//log.Println("field: cannot pass to dst", dst, "- diagonal move with blocking siblings")
			return PS_IMPASSABLE
		}
	}
	return PS_PASSABLE
}

func (f *Field) MoveMe(id int, coord UnitCoord) UnitCoord {
	f.units[id].coord = coord
	return coord
}

func (f *Field) KillMe(id int) {
	// kill unit
	f.units[id].agent.DetachUnit(f.units[id].unit)
	unit := f.units[id].unit
	f.units[id] = UnitPresence{coord: f.units[id].coord, agent: nopAgent,
		unit: &Corpse{f, id, unit, 0}}
}

func (f *Field) FindPath(from, to CellCoord) Path {
	finder := NewPathFinder(f)
	path := finder.FindPath(from, to)
	f.pathfinder = finder //FIXME(pathfind): remove after debug
	return path
}

// terrain api
// makePassableField makes everything but border passable
func (f *Field) makePassableField() {
	for i := 1; i < f.xSize-1; i++ {
		for j := 1; j < f.ySize-1; j++ {
			f.CellAt(CellCoord{i, j}).passable = true
		}
	}
}

func (f *Field) computeSlopes() {
	// slope is made when elevation level of adjacent cell is greater by 1 from current cell
	for i := 0; i < f.xSize-1; i++ {
		for j := 0; j < f.ySize-1; j++ {
			// compare cell with right and down neighbours
			coord := CellCoord{i, j}
			cell := f.CellAt(coord)
			right := f.CellAt(coord.Add(1, 0))
			down := f.CellAt(coord.Add(0, 1))

			switch cell.elevation - right.elevation {
			case 1:
				right.slopes |= SLOPE_LEFT
			case -1:
				cell.slopes |= SLOPE_RIGHT
			}

			switch cell.elevation - down.elevation {
			case 1:
				down.slopes |= SLOPE_UP
			case -1:
				cell.slopes |= SLOPE_DOWN
			}
		}
	}
}

type UnitPresence struct {
	coord UnitCoord
	agent Agent
	unit  Unit
}

type Cell struct {
	elevation int16
	slopes    uint8
	passable  bool
	object    Object
	items     []Item
}
