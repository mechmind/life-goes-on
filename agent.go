package main

import (
	"math/rand"
)

const (
	DAMSEL_WANDER_RADIUS  = 40
	DAMSEL_WANDER_TRIES   = 3
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
	Units       []*Soldier
	Target      UnitCoord
	Automove    bool
	Orders      chan Order
	FireState   int
	GrenTo      CellCoord
	GrenTimeout int
	Pid         int
}

func (s *Squad) AttachUnit(u Unit) {
	s.Units = append(s.Units, u.(*Soldier))
}

func (s *Squad) DetachUnit(u Unit) {
	dsol := u.(*Soldier)
	oldLen := len(s.Units)
	for idx, sol := range s.Units {
		if dsol == sol {
			// remove unit from slice
			s.Units = append(s.Units[:idx], s.Units[idx+1:]...)
			s.Units[:oldLen][oldLen-1] = nil
			return
		}
	}
}

func (s *Squad) HandleUnit(f *FieldView, u Unit, Coord UnitCoord) {
	soldier := u.(*Soldier)

	if soldier.SemifireCounter > 0 {
		soldier.SemifireCounter--
	}

	if s.GrenTo != (CellCoord{0, 0}) && s.GrenTimeout == 0 {
		GrenTo := s.GrenTo.UnitCenter()
		if Coord.Distance(GrenTo) < SOL_GREN_RANGE && f.HaveLOS(Coord, GrenTo) {
			// throw gren
			s.GrenTo = CellCoord{0, 0}
			f.ThrowGren(Coord, GrenTo)
			s.GrenTimeout = SOL_GREN_TIMEOUT
			return
		}
	}

	if s.FireState == ORDER_FIRE ||
		(s.FireState == ORDER_SEMIFIRE && soldier.SemifireCounter == 0) {
		byDistance := f.UnitsInRange(Coord, soldier.Gunner.FireRange)
		// check any zeds nearby
		for _, zed := range byDistance {
			if _, ok := zed.Unit.(*Zed); ok {
				if soldier.CanShoot(Coord, zed.Coord) {
					// shoot that zed
					soldier.Shoot(Coord, zed.Coord, zed.Unit)
					soldier.SemifireCounter = SOL_SEMIFIRE_TICKS
					return
				}
			}
		}
	}

	if s.Target.Cell() == (CellCoord{0, 0}) {
		return
	}
	// no zeds in fire range, move toward target
	if s.Target != soldier.Target {
		soldier.Target = s.Target
		// select nearby cell, to not jam entire squad into one
		sid := soldier.Id % 4
		lx := sid % 2
		ly := sid / 2
		myTargetCell := s.Target.Cell().Add(lx, ly)
		if f.field.CellAt(myTargetCell).Passable {
			soldier.MyTarget = myTargetCell.UnitCenter()
		} else {
			soldier.MyTarget = s.Target
		}
		soldier.path = f.FindPath(Coord.Cell(), soldier.MyTarget.Cell())
	}

	Target, ok := soldier.path.Current()
	if ok {
		if Coord.Distance(Target.UnitCenter()) < FLOAT_ERROR {
			Target, ok = soldier.path.Next()
			if ok {
				u.MoveToward(Coord, Target.UnitCenter())
			}
		} else {
			u.MoveToward(Coord, Target.UnitCenter())
		}
	}
}

func (s *Squad) Think(view *FieldView, tick int64) {
	if len(s.Units) == 0 {
		return
	}

	// read orders
OrderLoop:
	for {
		select {
		case order := <-s.Orders:
			switch order.order {
			case ORDER_MOVE:
				s.Target = order.Coord.UnitCenter()
				s.Automove = false
			case ORDER_AUTOMOVE:
				s.Automove = true

			case ORDER_FIRE:
				fallthrough
			case ORDER_SEMIFIRE:
				fallthrough
			case ORDER_NOFIRE:
				s.FireState = order.order

			case ORDER_GREN:
				s.GrenTo = order.Coord
			}
		default:
			break OrderLoop
		}
	}

	if s.Automove {
		Coord, _ := view.UnitByID(s.Units[0].GetID())
		// make path to nearby zed
		if tick%SQUAD_RETARGET_TICKS == 0 {
			byDistance := view.UnitsByDistance(Coord)
			var zed UnitPresence
			var zedFound bool
			for Id, u := range byDistance {
				if _, ok := u.Unit.(*Zed); ok {
					zed = byDistance[Id]
					zedFound = true
					break
				}
			}

			if !zedFound {
				// nothing to do
				return
			}

			// chase toward zed
			s.Target = zed.Coord
		}
	}

	if s.GrenTimeout > 0 {
		s.GrenTimeout--
	}
}

type ZedSwarm struct {
	Units []*Zed
}

func (z *ZedSwarm) AttachUnit(u Unit) {
	z.Units = append(z.Units, u.(*Zed))
}

func (z *ZedSwarm) DetachUnit(u Unit) {
	// FIXME: implement
}

func (z *ZedSwarm) HandleUnit(f *FieldView, u Unit, Coord UnitCoord) {
	var zed *Zed
	switch u.(type) {
	case *Zed:
		zed = u.(*Zed)
	case *Corpse:
		corpse := u.(*Corpse)
		corpse.RessurectCounter--
		if corpse.RessurectCounter == 0 {
			// respawn corpse as fresh new zed
			f.ReplaceUnit(corpse.Id, z, corpse.Respawn())
		}
		return
	}

	if !zed.Digest() {
		// starved
		return
	}

	var Target UnitCoord
	if zed.LastAttacker >= 0 {
		// fight back
		attackerCoord, attacker := f.UnitByID(zed.LastAttacker)
		if zed.CanBite(Coord, attackerCoord) {
			zed.Bite(Coord, attackerCoord, attacker)
			_, attacker = f.UnitByID(zed.LastAttacker)
			if corpse, ok := attacker.(*Corpse); ok {
				// foe is bitten to death
				zed.LastAttacker = -1
				if zed.Nutrition > ZED_NUTRITION_FULL {
					// infect corpse
					corpse.RessurectCounter = CORPSE_RESSURECT_TICKS
					// regain control over it
					f.Reown(corpse.Id, z)
					zed.Eat(ZED_INFECT_NUTRITION)
				} else {
					// eat it
					zed.Eat(ZED_EAT_NUTRITION)
				}
			}
		} else {
			Target = attackerCoord
		}
	} else {
		// find nearby human and attack it
		byDistance := f.UnitsByDistance(Coord)
		var nonzed UnitPresence
		var nonzedFound bool
		for Id, u := range byDistance {
			if _, ok := u.Unit.(*Zed); !ok {
				if _, ok := u.Unit.(*Corpse); !ok {
					nonzed = byDistance[Id]
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
		dest := nonzed.Coord
		if zed.CanBite(Coord, dest) {
			zed.Bite(Coord, dest, nonzed.Unit)
			_, victim := f.UnitByID(nonzed.Unit.GetID())
			if corpse, ok := victim.(*Corpse); ok {
				// victim is bitten to death, eat it
				if zed.Nutrition > ZED_NUTRITION_FULL {
					// infect corpse
					corpse.RessurectCounter = CORPSE_RESSURECT_TICKS
					// regain control over it
					f.Reown(corpse.Id, z)
					zed.Eat(ZED_INFECT_NUTRITION)
				} else {
					// eat it
					zed.Eat(ZED_EAT_NUTRITION)
				}
				return
			}
		} else {
			Target = dest
		}
	}

	if f.HaveLOS(Coord, Target) {
		// forget about path, rush toward target
		zed.path = nil
		zed.MoveToward(Coord, Target)
	} else if Target.Cell() != (CellCoord{0, 0}) {
		// follow path to target or create one
		if zed.path == nil {
			zed.path = f.FindPath(Coord.Cell(), Target.Cell())
		}
		Target, ok := zed.path.Current()
		if ok {
			if Coord.Distance(Target.UnitCenter()) < FLOAT_ERROR {
				Target, ok = zed.path.Next()
				if ok {
					u.MoveToward(Coord, Target.UnitCenter())
				}
			} else {
				u.MoveToward(Coord, Target.UnitCenter())
			}
		} else {
			zed.path = nil
		}
	}
}

type DamselCrowd struct {
	Units []*Damsel
}

func (d *DamselCrowd) AttachUnit(u Unit) {
	d.Units = append(d.Units, u.(*Damsel))
}

func (d *DamselCrowd) DetachUnit(u Unit) {
	// FIXME: implement
}

func (d *DamselCrowd) HandleUnit(f *FieldView, u Unit, Coord UnitCoord) {
	dam := u.(*Damsel)
	if dam.LastAttacker >= 0 {
		// flee away
		attackerCoord, attacker := f.UnitByID(dam.LastAttacker)
		dam.MoveAway(Coord, attackerCoord)
		if _, ok := attacker.(*Corpse); ok {
			// attacker is dead, 'calm' down
			dam.LastAttacker = -1
		}
	} else if dam.Adrenaline > 0 {
		// flee from panic point
		dam.MoveAway(Coord, dam.PanicPoint)
	} else {
		if dam.WanderTarget == Coord {
			// wander around
			for i := 0; i < DAMSEL_WANDER_TRIES; i++ {
				rx := ibound(Coord.Cell().X+int(rand.Int31n(DAMSEL_WANDER_RADIUS))-
					DAMSEL_WANDER_RADIUS/2, 0, 1024)
				ry := ibound(Coord.Cell().Y+int(rand.Int31n(DAMSEL_WANDER_RADIUS))-
					DAMSEL_WANDER_RADIUS/2, 0, 1024)
				newCoord := CellCoord{rx, ry}.UnitCenter()
				if f.HaveLOS(Coord, newCoord) {
					dam.WanderTarget = newCoord
				}
			}
		}
		dam.MoveToward(Coord, dam.WanderTarget)
	}

	dam.Adrenaline -= DAM_ADRENALINE_FADE
	if dam.Adrenaline < 0 {
		dam.Adrenaline = 0
		if dam.PanicPoint != (UnitCoord{0, 0}) {
			dam.WanderTarget = Coord
			dam.PanicPoint = UnitCoord{0, 0}
		}
	}
}

type NopAgent struct{}

func (n NopAgent) AttachUnit(u Unit)                                {}
func (n NopAgent) DetachUnit(u Unit)                                {}
func (n NopAgent) HandleUnit(f *FieldView, u Unit, Coord UnitCoord) {}
