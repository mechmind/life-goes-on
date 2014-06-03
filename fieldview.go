package main

import (
	"sort"
)

type FieldView struct {
	field *Field
}

func (f *FieldView) UnitsByDistance(src UnitCoord) []UnitPresence {
	presence := make([]UnitPresence, len(f.field.Units))
	copy(presence, f.field.Units)
	sorter := &unitsByDistance{src, presence}
	sort.Sort(sorter)
	return sorter.Units
}

func (f *FieldView) UnitsInRange(src UnitCoord, r float32) []UnitPresence {
	presence := make([]UnitPresence, 0)
	for _, u := range f.field.Units {
		if src.Distance(u.Coord) < r {
			presence = append(presence, u)
		}
	}
	sorter := &unitsByDistance{src, presence}
	sort.Sort(sorter)
	return sorter.Units
}

func (f *FieldView) UnitByID(Id int) (UnitCoord, Unit) {
	return f.field.UnitByID(Id)
}

func (f *FieldView) Reown(Id int, Agent Agent) {
	f.field.Units[Id].Agent = Agent
}

func (f *FieldView) ReplaceUnit(Id int, Agent Agent, u Unit) {
	Coord := f.field.Units[Id].Coord
	f.field.ReplaceUnit(Id, Coord, Agent, u)
}

func (f *FieldView) FindPath(From, To CellCoord) Path {
	return f.field.FindPath(From, To)
}

func (f *FieldView) HaveLOS(From, To UnitCoord) bool {
	return f.field.HaveLOS(From, To)
}

func (f *FieldView) ThrowGren(From, To UnitCoord) {
	f.field.ThrowGren(From, To)
}

// unitsByDistance used to sort units on field, nearest to src first
type unitsByDistance struct {
	src   UnitCoord
	Units []UnitPresence
}

// sort.Interface implementation
func (u *unitsByDistance) Len() int {
	return len(u.Units)
}

func (u *unitsByDistance) Less(i, j int) bool {
	return u.src.Distance(u.Units[i].Coord) < u.src.Distance(u.Units[j].Coord)
}

func (u *unitsByDistance) Swap(i, j int) {
	u.Units[i], u.Units[j] = u.Units[j], u.Units[i]
}
