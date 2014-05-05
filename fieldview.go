package main

import (
	"sort"
)

type FieldView struct {
	field *Field
}

func (f *FieldView) UnitsByDistance(src UnitCoord) []UnitPresence {
	presence := make([]UnitPresence, len(f.field.units))
	copy(presence, f.field.units)
	sorter := &unitsByDistance{src, presence}
	sort.Sort(sorter)
	return sorter.units
}

func (f *FieldView) UnitByID(id int) (UnitCoord, Unit) {
	return f.field.units[id].coord, f.field.units[id].unit
}

func (f *FieldView) Reown(id int, agent Agent) {
	f.field.units[id].agent = agent
}

func (f *FieldView) ReplaceUnit(id int, agent Agent, u Unit) {
	coord := f.field.units[id].coord
	f.field.ReplaceUnit(id, coord, agent, u)
}

// unitsByDistance used to sort units on field, nearest to src first
type unitsByDistance struct {
	src   UnitCoord
	units []UnitPresence
}

// sort.Interface implementation
func (u *unitsByDistance) Len() int {
	return len(u.units)
}

func (u *unitsByDistance) Less(i, j int) bool {
	return u.src.Distance(u.units[i].coord) < u.src.Distance(u.units[j].coord)
}

func (u *unitsByDistance) Swap(i, j int) {
	u.units[i], u.units[j] = u.units[j], u.units[i]
}
