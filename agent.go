package main

import (
	//"fmt"
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
		//fmt.Println("[squad] no task for soldier", u.(*Soldier).id)
		return
	}

	// chase toward zed
	dest := zed.coord
	soldier := u.(*Soldier)
	if soldier.CanShoot(coord, dest) {
		//fmt.Println("[squad] soldier", u.(*Soldier).id, "shooting at zed", zed.unit.(*Zed).id,
		//	"at", dest)
		soldier.Shoot(coord, dest, zed.unit)
	} else {
		u.MoveToward(coord, dest)
		//fmt.Println("[squad] moved soldier", u.(*Soldier).id, "from", coord, "to", to)
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
			//fmt.Println("[swarm] biting", attacker.GetID())
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
			//fmt.Println("[swarm] moved zed", u.(*Zed).id, coord,
			//	"==>", attackerCoord)
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
			//fmt.Println("[swarm] no humans for zed", zed.id)
			return
		}

		// chase toward nonzed
		dest := nonzed.coord
		if zed.CanBite(coord, dest) {
			//fmt.Println("[swarm] biting", u.GetID())
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
			//fmt.Println("[swarm] moved zed", zed.id, coord, "->", dest, "n:", zed.nutrition,
			//"r:", zed.rage, "s:", zed.Walker.WalkSpeed, "h:", zed.health)
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
		attackerCoord, _ := f.UnitByID(dam.lastAttacker)
		dam.MoveAway(coord, attackerCoord)
		//fmt.Println("[dam] fleeing dam", dam.id, "from", coord, "to", to)
	} else {
		if dam.wanderTarget == coord {
			// wander around
			rx := fbound(rand.Float32()*DAMSEL_WANDER_RADIUS-DAMSEL_WANDER_RADIUS/2, 0, 1024)
			ry := fbound(rand.Float32()*DAMSEL_WANDER_RADIUS-DAMSEL_WANDER_RADIUS/2, 0, 1024)
			dest := coord.Add(rx, ry)
			dam.wanderTarget = dest
			//fmt.Println("[dam] selected wandering target for", dam.id, "to", dest)
		}
		dam.MoveToward(coord, dam.wanderTarget)
		//fmt.Println("[dam] moved dam", dam.id, "from", coord, "to", dam.wanderTarget)
	}
}

type NopAgent struct{}

func (n NopAgent) AttachUnit(u Unit)                                {}
func (n NopAgent) DetachUnit(u Unit)                                {}
func (n NopAgent) HandleUnit(f *FieldView, u Unit, coord UnitCoord) {}
