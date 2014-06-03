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
		if src.Distance(u.coord) < r {
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

func (f *FieldView) Reown(Id int, agent Agent) {
	f.field.Units[Id].agent = agent
}

func (f *FieldView) ReplaceUnit(Id int, agent Agent, u Unit) {
	coord := f.field.Units[Id].coord
	f.field.ReplaceUnit(Id, coord, agent, u)
}

func (f *FieldView) FindPath(from, to CellCoord) Path {
	return f.field.FindPath(from, to)
}

func (f *FieldView) HaveLOS(from, to UnitCoord) bool {
	return f.field.HaveLOS(from, to)
}

func (f *FieldView) ThrowGren(from, to UnitCoord) {
	f.field.ThrowGren(from, to)
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
	return u.src.Distance(u.Units[i].coord) < u.src.Distance(u.Units[j].coord)
}

func (u *unitsByDistance) Swap(i, j int) {
	u.Units[i], u.Units[j] = u.Units[j], u.Units[i]
}
