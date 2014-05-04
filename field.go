package main

import (
	"fmt"
)

const (
	FLOAT_ERROR = 0.000001
)

var nopAgent NopAgent

type Field struct {
	xSize, ySize int
	cells        []Cell
	agents       []Agent
	units        []UnitPresence
}

func NewField(xSize, ySize int) *Field {
	return &Field{xSize, ySize, make([]Cell, xSize*ySize), nil, nil}
}

func (f *Field) Tick(tick int64) {
	fmt.Println("[field] tick")
	//for _, agent := range f.agents {
	//	agent.Tick(tick)
	//}

	view := &FieldView{f}
	for _, up := range f.units {
		up.agent.HandleUnit(view, up.unit, up.coord)
	}
}

func (f *Field) PlaceUnit(c UnitCoord, agent Agent, u Unit) error {
	f.units = append(f.units, UnitPresence{c, agent, u})
	f.units[len(f.units)-1].unit.SetID(len(f.units)-1)
	agent.AttachUnit(u)
	return nil
}

func (f *Field) PlaceAgent(a Agent) error {
	f.agents = append(f.agents, a)
	return nil
}

// unit/agent api
func (f *Field) CellAt(c CellCoord) *Cell {
	return &f.cells[c.Y*f.xSize+c.X]
}

func (f *Field) MoveMe(id int, coord UnitCoord) UnitCoord {
	f.units[id].coord = coord
	return coord
}

func (f *Field) KillMe(id int) {
	// kill unit
	f.units[id].agent.DetachUnit(f.units[id].unit)
	f.units[id] = UnitPresence{coord: f.units[id].coord, agent: nopAgent, unit: nil}
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
