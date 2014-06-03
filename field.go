package main

import (
	"math/rand"
	"time"
)

const (
	FLOAT_ERROR           = 0.000001
	FIELD_BACKBUFFER_SIZE = 3
	FIELD_SIZE            = 1024
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
	XSize, YSize int
	Cells        []Cell
	Agents       []Agent
	Units        []UnitPresence
	updates      chan *Field

	// FIXME(pathfind): remove after debugging
	pathfinder *PathFinder

	// moving stuff
	Grens []FlyingGren

	// rng
	rng *rand.Rand

	// game state
	gameState chan GameState
}

func NewField(XSize, YSize int, updates chan *Field) *Field {
	rng := rand.New(rand.NewSource(time.Now().Unix()))
	field := &Field{XSize, YSize, make([]Cell, XSize*YSize), nil, nil, updates, nil, nil, rng,
		make(chan GameState, 5)}
	field.makePassableField()
	field.computeSlopes()
	return field
}

func copyField(f *Field) *Field {
	bb := &Field{XSize: f.XSize, YSize: f.YSize}

	bb.Cells = f.Cells
	bb.Units = append(bb.Units[:0], f.Units...)
	bb.Agents = append(bb.Agents[:0], f.Agents...)
	bb.Grens = append(bb.Grens[:0], f.Grens...)
	//bb.pathfinder = f.pathfinder // FIXME(pathfind)

	return bb
}

func (f *Field) Tick(tick int64) {
	view := &FieldView{f}

	for _, agent := range f.Agents {
		if thinker, ok := agent.(Thinker); ok {
			thinker.Think(view, tick)
		}
	}

	for _, up := range f.Units {
		up.agent.HandleUnit(view, up.Unit, up.coord)
	}

	// handle exploded grens
	for i := 0; i < len(f.Grens); {
		if f.Grens[i].booming == SOL_GREN_TICK_CAP {
			// remove that gren
			copy(f.Grens[i:], f.Grens[i+1:])
			f.Grens = f.Grens[:len(f.Grens)-1]
			continue
		} else if f.Grens[i].booming > 0 {
			f.Grens[i].booming++
		}
		i++
	}

	// handle flying grens
	for idx, gren := range f.Grens {
		toward := NormTowardCoord(gren.from, gren.to).Mult(SOL_GREN_SPEED)
		if gren.from.Distance(gren.to) < FLOAT_ERROR && f.Grens[idx].booming == 0 {
			// BOOM
			f.Grens[idx].booming = 1 // for animation
			for _, u := range view.UnitsInRange(gren.to, SOL_GREN_RADIUS) {
				if f.HaveLOS(u.coord, gren.to) {
					u.Unit.RecieveDamage(-1, SOL_GREN_DAMAGE)
				}
			}
		}
		if gren.from.Distance(gren.to) < toward.Distance(UnitCoord{0, 0}) {
			f.Grens[idx].from = gren.to
		} else {
			f.Grens[idx].from = gren.from.AddCoord(toward)
		}
	}

	// check game over
	if tick%TIME_TICKS_PER_SEC == 0 {
		var Zs, Bs, Ss int
		for _, u := range f.Units {
			switch u.Unit.(type) {
			case *Zed:
				Zs++
			case *Damsel:
				Bs++
			case *Soldier:
				Ss++
			}
		}
		if Ss == 0 {
			f.gameState <- GameState{GAME_OVER, -1}
		}
		// TODO: rebuild
		/*
			if Zs == 0 {
				if Bs == 0 {
					f.gameState <- GAME_DRAW
				} else {
					f.gameState <- GAME_WIN
				}
			}
			//*/
	}

	// send update
	select {
	case f.updates <- copyField(f):
	default:
	}
}

func (f *Field) PlaceUnit(c UnitCoord, agent Agent, u Unit) error {
	f.Units = append(f.Units, UnitPresence{c, agent, u})
	u.SetID(len(f.Units) - 1)
	agent.AttachUnit(u)
	return nil
}

func (f *Field) PlaceAgent(a Agent) error {
	f.Agents = append(f.Agents, a)
	return nil
}

func (f *Field) ReplaceUnit(Id int, c UnitCoord, agent Agent, u Unit) error {
	f.Units[Id] = UnitPresence{c, agent, u}
	u.SetID(Id)
	agent.AttachUnit(u)
	return nil
}

// unit/agent api
func (f *Field) CellAt(c CellCoord) *Cell {
	return &f.Cells[c.Y*f.XSize+c.X]
}

func (f *Field) UnitByID(Id int) (UnitCoord, Unit) {
	return f.Units[Id].coord, f.Units[Id].Unit
}

func (f *Field) UnitsInRange(center UnitCoord, radius float32) []UnitPresence {
	var Units []UnitPresence

	for _, up := range f.Units {
		if center.Distance(up.coord) < radius {
			Units = append(Units, up)
		}
	}
	return Units
}

// return true if have line of sight from 'from' to 'to'
func (f *Field) TraceShot(from, to UnitCoord, tid int) (atid int, atcoord UnitCoord) {
	// misshots start when accuracy starting do decay
	if from.Distance(to) <= SOL_ACC_DECAY_START {
		return tid, to
	}

	toward := NormTowardCoord(from, to)

	low := UnitCoord{fmin(from.X, to.X), fmin(from.Y, to.Y)}.Cell()
	high := UnitCoord{fmax(from.X, to.X), fmax(from.Y, to.Y)}.Cell()

	Units := make(map[CellCoord][]UnitPresence)
	for _, up := range f.Units {
		cellCoord := up.coord.Cell()
		if CheckCellCoordBounds(cellCoord, low, high) {
			us := append(Units[cellCoord], up)
			Units[cellCoord] = us
		}
	}

	current := from.AddCoord(toward.Mult(SOL_ACC_DECAY_START))
	for {
		// always check next and current cell passability because we can advance 2 cells
		// on one step
		if f.CellAt(current.Cell()).passable == false {
			return -1, current
		}

		currentCell := current.Cell()
		unitsThere := Units[currentCell]
		for _, u := range unitsThere {
			if f.rng.Intn(100) < SOL_MISSHOT_PROB {
				// misshot
				return u.Unit.GetID(), u.coord
			}
		}

		stepCoord := NextCellCoord(current, toward)
		if currentCell.AddCoord(stepCoord) != current.AddCoord(toward).Cell() {
			unitsHere := Units[currentCell.AddCoord(stepCoord)]
			for _, u := range unitsHere {
				if f.rng.Intn(100) < SOL_MISSHOT_PROB {
					// misshot
					return u.Unit.GetID(), u.coord
				}
			}
		}

		if current == to {
			return tid, to
		}

		if current.Distance(to) < 1 {
			current = to
		} else {
			current = current.AddCoord(toward)
		}
	}
	//return tid, to
}

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

func (f *Field) MoveMe(Id int, coord UnitCoord) UnitCoord {
	f.Units[Id].coord = coord
	return coord
}

func (f *Field) KillMe(Id int) {
	// kill unit
	f.Units[Id].agent.DetachUnit(f.Units[Id].Unit)
	Unit := f.Units[Id].Unit
	f.Units[Id] = UnitPresence{coord: f.Units[Id].coord, agent: nopAgent,
		Unit: &Corpse{f, Id, Unit, 0}}
}

func (f *Field) ThrowGren(from, to UnitCoord) {
	f.Grens = append(f.Grens, FlyingGren{from, to, 0})
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
	for i := 1; i < f.XSize-1; i++ {
		for j := 1; j < f.YSize-1; j++ {
			f.CellAt(CellCoord{i, j}).passable = true
		}
	}
}

func (f *Field) computeSlopes() {
	// slope is made when elevation level of adjacent cell is greater by 1 from current cell
	for i := 0; i < f.XSize-1; i++ {
		for j := 0; j < f.YSize-1; j++ {
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
	Unit  Unit
}

type Cell struct {
	elevation int16
	slopes    uint8
	passable  bool
	object    Object
	items     []Item
}

type FlyingGren struct {
	from, to UnitCoord
	booming  int8
}
