package main

import (
	"fmt"
)

type Agent interface {
	AttachUnit(Unit)
	DetachUnit(Unit)

	HandleUnit(*FieldView, Unit, UnitCoord)
}

type Squad struct {
	units  []*Soldier
	moveTo CellCoord
}

func (s *Squad) AttachUnit(u Unit) {
	s.units = append(s.units, u.(*Soldier))
}

func (s *Squad) DetachUnit(u Unit) {
	// FIXME: implement
}

func (s *Squad) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {
	// find nearby zed and move toward it
	byDistance := f.UnitsByDistance(coord)
	var zed UnitPresence
	var zedFound bool
	for id, u := range byDistance {
		if _, ok := u.unit.(*Zed); ok {
			zed = byDistance[id]
			zedFound = true
			break
		}
	}

	if ! zedFound  {
		// nothing to do
		fmt.Println("[squad] no task for soldier", u.(*Soldier).id)
		return
	}

	// chase toward zed
	dest := zed.coord
	soldier := u.(*Soldier)
	if soldier.CanShoot(coord, dest) {
		fmt.Println("[squad] soldier", u.(*Soldier).id, "shooting at zed", zed.unit.(*Zed).id,
			"at", dest)
		soldier.Shoot(coord, dest, zed.unit)
	} else {
		to := u.MoveToward(coord, dest)
		fmt.Println("[squad] moved soldier", u.(*Soldier).id, "from", coord, "to", to)
	}
}

type ZedSwarm struct {
	units []*Zed
}

func (z *ZedSwarm) AttachUnit(u Unit) {
	z.units = append(z.units, u.(*Zed))
}

func (z *ZedSwarm) DetachUnit(u Unit) {
	// FIXME: implement
}

func (z *ZedSwarm) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {
	dest := UnitCoord{0, 100}
	to := u.MoveToward(coord, dest)
	fmt.Println("[swarm] moved zed", u.(*Zed).id, "from", coord, "to", to)
}

type NopAgent struct{}

func (n NopAgent) AttachUnit(u Unit) {}
func (n NopAgent) DetachUnit(u Unit) {}
func (n NopAgent) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {}
