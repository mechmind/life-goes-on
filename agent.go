package main

import (
	"math/rand"
)

const DAMSEL_WANDER_RADIUS = 40

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

	if !zedFound {
		// nothing to do
		return
	}

	// chase toward zed
	dest := zed.coord
	soldier := u.(*Soldier)
	if soldier.CanShoot(coord, dest) {
		//	"at", dest)
		soldier.Shoot(coord, dest, zed.unit)
	} else {
		u.MoveToward(coord, dest)
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
			rx := fbound(coord.X+rand.Float32()*DAMSEL_WANDER_RADIUS-DAMSEL_WANDER_RADIUS/2,
				0, 1024)
			ry := fbound(coord.Y+rand.Float32()*DAMSEL_WANDER_RADIUS-DAMSEL_WANDER_RADIUS/2,
				0, 1024)
			dam.wanderTarget = UnitCoord{rx, ry}
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
