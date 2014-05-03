package main

import (
	"fmt"
)

type Agent interface {
	Ticker
	AttachUnit(Unit)
	DetachUnit(Unit)

	HandleUnit(Unit, UnitCoord)
}

type Squad struct {
	units []*Soldier
	moveTo CellCoord
}

func (s *Squad) Tick(tick int64) {

}

func (s *Squad) AttachUnit(u Unit) {
	s.units = append(s.units, u.(*Soldier))
}

func (s *Squad) DetachUnit(u Unit) {
	// FIXME: implement
}

func (s *Squad) HandleUnit(u Unit, coord UnitCoord) {
	// just move it to (100, 100)
	dest := UnitCoord{100, 100}
	fmt.Println("[agent] moving unit", u.(*Soldier).id, "from", coord, "toward", dest)
	to := u.MoveToward(coord, dest)
	fmt.Println("[agent] moved unit", u.(*Soldier).id, "from", coord, "to", to)
}

type Swarm struct {
	units []Unit
}
