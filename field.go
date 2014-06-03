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

	for _, Agent := range f.Agents {
		if thinker, ok := Agent.(Thinker); ok {
			thinker.Think(view, tick)
		}
	}

	for _, up := range f.Units {
		up.Agent.HandleUnit(view, up.Unit, up.Coord)
	}

	// handle exploded grens
	for i := 0; i < len(f.Grens); {
		if f.Grens[i].Booming == SOL_GREN_TICK_CAP {
			// remove that gren
			copy(f.Grens[i:], f.Grens[i+1:])
			f.Grens = f.Grens[:len(f.Grens)-1]
			continue
		} else if f.Grens[i].Booming > 0 {
			f.Grens[i].Booming++
		}
		i++
	}

	// handle flying grens
	for idx, gren := range f.Grens {
		toward := NormTowardCoord(gren.From, gren.To).Mult(SOL_GREN_SPEED)
		if gren.From.Distance(gren.To) < FLOAT_ERROR && f.Grens[idx].Booming == 0 {
			// BOOM
			f.Grens[idx].Booming = 1 // for animation
			for _, u := range view.UnitsInRange(gren.To, SOL_GREN_RADIUS) {
				if f.HaveLOS(u.Coord, gren.To) {
					u.Unit.RecieveDamage(-1, SOL_GREN_DAMAGE)
				}
			}
		}
		if gren.From.Distance(gren.To) < toward.Distance(UnitCoord{0, 0}) {
			f.Grens[idx].From = gren.To
		} else {
			f.Grens[idx].From = gren.From.AddCoord(toward)
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

func (f *Field) PlaceUnit(c UnitCoord, Agent Agent, u Unit) error {
	f.Units = append(f.Units, UnitPresence{c, Agent, u})
	u.SetID(len(f.Units) - 1)
	Agent.AttachUnit(u)
	return nil
}

func (f *Field) PlaceAgent(a Agent) error {
	f.Agents = append(f.Agents, a)
	return nil
}

func (f *Field) ReplaceUnit(Id int, c UnitCoord, Agent Agent, u Unit) error {
	f.Units[Id] = UnitPresence{c, Agent, u}
	u.SetID(Id)
	Agent.AttachUnit(u)
	return nil
}

// unit/agent api
func (f *Field) CellAt(c CellCoord) *Cell {
	return &f.Cells[c.Y*f.XSize+c.X]
}

func (f *Field) UnitByID(Id int) (UnitCoord, Unit) {
	return f.Units[Id].Coord, f.Units[Id].Unit
}

func (f *Field) UnitsInRange(center UnitCoord, radius float32) []UnitPresence {
	var Units []UnitPresence

	for _, up := range f.Units {
		if center.Distance(up.Coord) < radius {
			Units = append(Units, up)
		}
	}
	return Units
}

// return true if have line of sight from 'from' to 'to'
func (f *Field) TraceShot(From, To UnitCoord, tid int) (atid int, atcoord UnitCoord) {
	// misshots start when accuracy starting do decay
	if From.Distance(To) <= SOL_ACC_DECAY_START {
		return tid, To
	}

	toward := NormTowardCoord(From, To)

	low := UnitCoord{fmin(From.X, To.X), fmin(From.Y, To.Y)}.Cell()
	high := UnitCoord{fmax(From.X, To.X), fmax(From.Y, To.Y)}.Cell()

	Units := make(map[CellCoord][]UnitPresence)
	for _, up := range f.Units {
		cellCoord := up.Coord.Cell()
		if CheckCellCoordBounds(cellCoord, low, high) {
			us := append(Units[cellCoord], up)
			Units[cellCoord] = us
		}
	}

	current := From.AddCoord(toward.Mult(SOL_ACC_DECAY_START))
	for {
		// always check next and current cell passability because we can advance 2 cells
		// on one step
		if f.CellAt(current.Cell()).Passable == false {
			return -1, current
		}

		currentCell := current.Cell()
		unitsThere := Units[currentCell]
		for _, u := range unitsThere {
			if f.rng.Intn(100) < SOL_MISSHOT_PROB {
				// misshot
				return u.Unit.GetID(), u.Coord
			}
		}

		stepCoord := NextCellCoord(current, toward)
		if currentCell.AddCoord(stepCoord) != current.AddCoord(toward).Cell() {
			unitsHere := Units[currentCell.AddCoord(stepCoord)]
			for _, u := range unitsHere {
				if f.rng.Intn(100) < SOL_MISSHOT_PROB {
					// misshot
					return u.Unit.GetID(), u.Coord
				}
			}
		}

		if current == To {
			return tid, To
		}

		if current.Distance(To) < 1 {
			current = To
		} else {
			current = current.AddCoord(toward)
		}
	}
	//return tid, to
}

func (f *Field) HaveLOS(From, To UnitCoord) bool {
	toward := NormTowardCoord(From, To)

	current := From
	for {
		// always check next and current cell passability because we can advance 2 cells
		// on one step
		if f.CellAt(current.Cell()).Passable == false {
			return false
		}

		if current == To {
			return true
		}

		nextCell := From.Cell().AddCoord(NextCellCoord(From, toward)).Bound(0, 0, 1024, 1024)
		if f.CellAt(nextCell).Passable == false {
			return false
		}

		if current.Distance(To) < 1 {
			current = To
		} else {
			current = current.AddCoord(toward)
		}
	}
}

func (f *Field) CheckPassability(src, dst CellCoord) Passability {
	dstCell := f.CellAt(dst)
	srcCell := f.CellAt(src)
	if !dstCell.Passable {
		//log.Println("field: cannot pass to dst", dst, "- is a wall")
		return PS_IMPASSABLE
	}
	if iabs(int(srcCell.Elevation-dstCell.Elevation)) > 2 {
		//log.Println("field: cannot pass to dst", dst, "- elevation is too high")
		return PS_IMPASSABLE
	}
	if src.Distance(dst) > 1 {
		// diagonal move
		direction := dst.AddCoord(src.Mult(-1))
		s1 := src.AddCoord(direction.ClockwiseSibling())
		s2 := src.AddCoord(direction.CounterclockwiseSibling())
		//log.Println("field: d:", direction, "siblings:", s1, s2)
		if !f.CellAt(s1).Passable || !f.CellAt(s2).Passable {
			//log.Println("field: cannot pass to dst", dst, "- diagonal move with blocking siblings")
			return PS_IMPASSABLE
		}
	}
	return PS_PASSABLE
}

func (f *Field) MoveMe(Id int, Coord UnitCoord) UnitCoord {
	f.Units[Id].Coord = Coord
	return Coord
}

func (f *Field) KillMe(Id int) {
	// kill unit
	f.Units[Id].Agent.DetachUnit(f.Units[Id].Unit)
	Unit := f.Units[Id].Unit
	f.Units[Id] = UnitPresence{Coord: f.Units[Id].Coord, Agent: nopAgent,
		Unit: &Corpse{f, Id, Unit, 0}}
}

func (f *Field) ThrowGren(From, To UnitCoord) {
	f.Grens = append(f.Grens, FlyingGren{From, To, 0})
}

func (f *Field) FindPath(From, To CellCoord) Path {
	finder := NewPathFinder(f)
	path := finder.FindPath(From, To)
	f.pathfinder = finder //FIXME(pathfind): remove after debug
	return path
}

// terrain api
// makePassableField makes everything but border passable
func (f *Field) makePassableField() {
	for i := 1; i < f.XSize-1; i++ {
		for j := 1; j < f.YSize-1; j++ {
			f.CellAt(CellCoord{i, j}).Passable = true
		}
	}
}

func (f *Field) computeSlopes() {
	// slope is made when elevation level of adjacent cell is greater by 1 from current cell
	for i := 0; i < f.XSize-1; i++ {
		for j := 0; j < f.YSize-1; j++ {
			// compare cell with right and down neighbours
			Coord := CellCoord{i, j}
			cell := f.CellAt(Coord)
			right := f.CellAt(Coord.Add(1, 0))
			down := f.CellAt(Coord.Add(0, 1))

			switch cell.Elevation - right.Elevation {
			case 1:
				right.Slopes |= SLOPE_LEFT
			case -1:
				cell.Slopes |= SLOPE_RIGHT
			}

			switch cell.Elevation - down.Elevation {
			case 1:
				down.Slopes |= SLOPE_UP
			case -1:
				cell.Slopes |= SLOPE_DOWN
			}
		}
	}
}

type UnitPresence struct {
	Coord UnitCoord
	Agent Agent
	Unit  Unit
}

type Cell struct {
	Elevation int16
	Slopes    uint8
	Passable  bool
}

type FlyingGren struct {
	From, To UnitCoord
	Booming  int8
}
