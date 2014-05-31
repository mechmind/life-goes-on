package main

import (
	"math/rand"
)

const (
	DAMSEL_WANDER_RADIUS  = 40
	SQUAD_RETARGET_TICKS  = 30
	SQUAD_ORDER_QUEUE_LEN = 16
)

type Agent interface {
	AttachUnit(Unit)
	DetachUnit(Unit)

	HandleUnit(*FieldView, Unit, UnitCoord)
}

type Thinker interface {
	Think(field *FieldView, tick int64)
}

type Squad struct {
	units  []*Soldier
	target UnitCoord
	automove bool
	orders chan Order
	fireState int
	grenTo CellCoord
}

func (s *Squad) AttachUnit(u Unit) {
	s.units = append(s.units, u.(*Soldier))
}

func (s *Squad) DetachUnit(u Unit) {
	dsol := u.(*Soldier)
	oldLen := len(s.units)
	for idx, sol := range s.units {
		if dsol == sol {
			// remove unit from slice
			s.units = append(s.units[:idx], s.units[idx+1:]...)
			s.units[:oldLen][oldLen-1] = nil
			return
		}
	}
}

func (s *Squad) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {
	soldier := u.(*Soldier)

	if soldier.semifireCounter > 0 {
		soldier.semifireCounter--
	}

	if (s.fireState == ORDER_FIRE ||
		(s.fireState == ORDER_SEMIFIRE && soldier.semifireCounter == 0)) {
		byDistance := f.UnitsInRange(coord, soldier.Gunner.fireRange)
		// check any zeds nearby
		for _, zed := range byDistance {
			if _, ok := zed.unit.(*Zed); ok {
				if soldier.CanShoot(coord, zed.coord) {
					// shoot that zed
					soldier.Shoot(coord, zed.coord, zed.unit)
					soldier.semifireCounter = SOL_SEMIFIRE_TICKS
					return
				}
			}
		}
	}

	if s.target.Cell() == (CellCoord{0, 0}) {
		return
	}
	// no zeds in fire range, move toward target
	if s.target != soldier.target {
		soldier.target = s.target
		soldier.path = f.FindPath(coord.Cell(), soldier.target.Cell())
	}

	target, ok := soldier.path.Current()
	if ok {
		if coord.Distance(target.UnitCenter()) < FLOAT_ERROR {
			target, ok = soldier.path.Next()
			if ok {
				u.MoveToward(coord, target.UnitCenter())
			}
		} else {
			u.MoveToward(coord, target.UnitCenter())
		}
	}
}

func (s *Squad) Think(view *FieldView, tick int64) {
	if len(s.units) == 0 {
		return
	}

	// read orders
OrderLoop:
	for {
		select {
		case order := <-s.orders:
			switch order.order {
			case ORDER_MOVE:
				s.target = order.coord.UnitCenter()
				s.automove = false
			case ORDER_AUTOMOVE:
				s.automove = true

			case ORDER_FIRE:
				fallthrough
			case ORDER_SEMIFIRE:
				fallthrough
			case ORDER_NOFIRE:
				s.fireState = order.order

			case ORDER_GREN:
				s.grenTo = order.coord
			}
		default:
			break OrderLoop
		}
	}

	if s.automove {
		coord, _ := view.UnitByID(s.units[0].GetID())
		// make path to nearby zed
		if tick%SQUAD_RETARGET_TICKS == 0 {
			byDistance := view.UnitsByDistance(coord)
			var zed UnitPresence
			var zedFound bool
			for id, u := range byDistance {
				if _, ok := u.unit.(*Zed); ok {
					zed = byDistance[id]
					zedFound = true
					break
				}
			}

			if !zedFound {
				// nothing to do
				return
			}

			// chase toward zed
			s.target = zed.coord
		}
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
	var zed *Zed
	switch u.(type) {
	case *Zed:
		zed = u.(*Zed)
	case *Corpse:
		corpse := u.(*Corpse)
		corpse.ressurectCounter--
		if corpse.ressurectCounter == 0 {
			// respawn corpse as fresh new zed
			f.ReplaceUnit(corpse.id, z, corpse.Respawn())
		}
		return
	}

	if !zed.Digest() {
		// starved
		return
	}

	if zed.lastAttacker >= 0 {
		// fight back
		attackerCoord, attacker := f.UnitByID(zed.lastAttacker)
		if zed.CanBite(coord, attackerCoord) {
			zed.Bite(coord, attackerCoord, attacker)
			_, attacker = f.UnitByID(zed.lastAttacker)
			if corpse, ok := attacker.(*Corpse); ok {
				// foe is bitten to death
				zed.lastAttacker = -1
				if zed.nutrition > ZED_NUTRITION_FULL {
					// infect corpse
					corpse.ressurectCounter = CORPSE_RESSURECT_TICKS
					// regain control over it
					f.Reown(corpse.id, z)
					zed.Eat(ZED_INFECT_NUTRITION)
				} else {
					// eat it
					zed.Eat(ZED_EAT_NUTRITION)
				}
			}
		} else {
			zed.MoveToward(coord, attackerCoord)
		}
	} else {
		// find nearby human and attack it
		byDistance := f.UnitsByDistance(coord)
		var nonzed UnitPresence
		var nonzedFound bool
		for id, u := range byDistance {
			if _, ok := u.unit.(*Zed); !ok {
				if _, ok := u.unit.(*Corpse); !ok {
					nonzed = byDistance[id]
					nonzedFound = true
					break
				}
			}
		}

		if !nonzedFound {
			// nothing to do
			return
		}

		// chase toward nonzed
		dest := nonzed.coord
		if zed.CanBite(coord, dest) {
			zed.Bite(coord, dest, nonzed.unit)
			_, victim := f.UnitByID(nonzed.unit.GetID())
			if corpse, ok := victim.(*Corpse); ok {
				// victim is bitten to death, eat it
				if zed.nutrition > ZED_NUTRITION_FULL {
					// infect corpse
					corpse.ressurectCounter = CORPSE_RESSURECT_TICKS
					// regain control over it
					f.Reown(corpse.id, z)
					zed.Eat(ZED_INFECT_NUTRITION)
				} else {
					// eat it
					zed.Eat(ZED_EAT_NUTRITION)
				}
			}
		} else {
			zed.MoveToward(coord, dest)
		}
	}
}

type DamselCrowd struct {
	units []*Damsel
}

func (d *DamselCrowd) AttachUnit(u Unit) {
	d.units = append(d.units, u.(*Damsel))
}

func (d *DamselCrowd) DetachUnit(u Unit) {
	// FIXME: implement
}

func (d *DamselCrowd) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {
	dam := u.(*Damsel)
	if dam.lastAttacker >= 0 {
		// flee away
		attackerCoord, attacker := f.UnitByID(dam.lastAttacker)
		dam.MoveAway(coord, attackerCoord)
		if _, ok := attacker.(*Corpse); ok {
			// attacker is dead, 'calm' down
			dam.lastAttacker = -1
		}
	} else if dam.adrenaline > 0 {
		// flee from panic point
		dam.MoveAway(coord, dam.panicPoint)
	} else {
		if dam.wanderTarget == coord {
			// wander around
			rx := ibound(coord.Cell().X+int(rand.Int31n(DAMSEL_WANDER_RADIUS))-
				DAMSEL_WANDER_RADIUS/2, 0, 1024)
			ry := ibound(coord.Cell().Y+int(rand.Int31n(DAMSEL_WANDER_RADIUS))-
				DAMSEL_WANDER_RADIUS/2, 0, 1024)
			dam.wanderTarget = CellCoord{rx, ry}.UnitCenter()
		}
		dam.MoveToward(coord, dam.wanderTarget)
	}

	dam.adrenaline -= DAM_ADRENALINE_FADE
	if dam.adrenaline < 0 {
		dam.adrenaline = 0
	}
}

type NopAgent struct{}

func (n NopAgent) AttachUnit(u Unit)                                {}
func (n NopAgent) DetachUnit(u Unit)                                {}
func (n NopAgent) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {}
