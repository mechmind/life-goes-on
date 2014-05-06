package main

import (
//"fmt"
)

const (
	FLOAT_ERROR           = 0.000001
	FIELD_BACKBUFFER_SIZE = 3
)

var nopAgent NopAgent

var fieldBackbuffer = make(chan *Field, FIELD_BACKBUFFER_SIZE)

type Field struct {
	xSize, ySize int
	cells        []Cell
	agents       []Agent
	units        []UnitPresence
	updates      chan *Field
}

func NewField(xSize, ySize int, updates chan *Field) *Field {
	return &Field{xSize, ySize, make([]Cell, xSize*ySize), nil, nil, updates}
}

func copyField(f *Field) *Field {
	var bb *Field
	select {
	case bb = <-fieldBackbuffer:
	default:
		bb = &Field{f.xSize, f.ySize, make([]Cell, f.xSize*f.ySize), nil, nil, nil}
	}

	copy(bb.cells, f.cells)
	bb.units = append(bb.units[:0], f.units...)
	bb.agents = append(bb.agents[:0], f.agents...)
	bb.updates = fieldBackbuffer

	return bb
}

func (f *Field) Tick(tick int64) {
	//fmt.Println("[field] tick")
	//for _, agent := range f.agents {
	//	agent.Tick(tick)
	//}

	view := &FieldView{f}
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

type UnitPresence struct {
	coord UnitCoord
	agent Agent
	unit  Unit
}

type Cell struct {
	elevation int16
	object    Object
	items     []Item
}
