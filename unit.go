package main

import (
	"fmt"
)

const (
	ZED_NUTRITION_WALKING = 1
	ZED_NUTRITION_BITING = 0.35
	ZED_RAGE_FROM_DAMAGE = 1
	ZED_RAGE_COOLING = 0.03
	ZED_RAGE_THRESHOLD = 3
	ZED_RAGE_COST = 0.1
	ZED_RAGE_SPEEDUP = 0.05
	ZED_RAGE_BITEUP = 0.3
	ZED_NUTRITION_TO_HP_PORTION = 5
	ZED_NUTRITION_TO_HP_THRESHOLD = 1200
	ZED_NUTRITION_TO_HP_SCALE = 0.02
	ZED_NUTRITION_FULL = 1600
	ZED_MOVER_WALK = 0.5
	ZED_MOVER_WALKUP = 0.3
	ZED_MOVER_WALKDOWN = 0.6
	ZED_EAT_NUTRITION = 350
	ZED_INFECT_NUTRITION = 50
	ZED_BITE_DAMAGE = 40
	ZED_HEALTH = 140
	ZED_NUTRITION_BASE = 1000

	CORPSE_RESSURECT_TICKS = 30
)


type Unit interface {
	SetID(int)
	GetID() int
	Mover
	DamageReciever
}

type Mover interface {
	MoveToward(src, dest UnitCoord) UnitCoord
}

type DamageReciever interface {
	RecieveDamage(from int, dmg float32)
}

type Walker struct {
	WalkSpeed     float32
	WalkUpSpeed   float32
	WalkDownSpeed float32
}

func (w *Walker) MoveToward(f *Field, src, dest UnitCoord) UnitCoord {
	toward := NormTowardCoord(src, dest)
	nextCellCoord := NextCellCoord(src, toward)
	currentCell := f.CellAt(src.Cell())
	nextCell := f.CellAt(nextCellCoord)

	//fmt.Printf("[unit] moving from %s toward %s: toward=%s, next cell=%s\n", src, dest,
	//	toward, nextCellCoord)
	var speed float32
	switch {
	case currentCell.elevation == nextCell.elevation:
		// walking straight
		speed = w.WalkSpeed
	case currentCell.elevation > nextCell.elevation:
		// walking down
		speed = w.WalkDownSpeed
	case currentCell.elevation < nextCell.elevation:
		// walking up
		speed = w.WalkUpSpeed
	}

	// advance
	nextPos := src.AddCoord(toward.Mult(speed))
	if src.Distance(nextPos) < src.Distance(dest) {
		return nextPos
	} else {
		return dest
	}
}

func (w *Walker) MoveAway(f *Field, src, dest UnitCoord) UnitCoord {
	toward := NormTowardCoord(src, dest)
	away := toward.Mult(-1)
	nextCellCoord := NextCellCoord(src, away)
	currentCell := f.CellAt(src.Cell())
	nextCell := f.CellAt(nextCellCoord)

	//fmt.Printf("[unit] moving from %s toward %s: toward=%s, next cell=%s\n", src, dest,
	//	toward, nextCellCoord)
	var speed float32
	switch {
	case currentCell.elevation == nextCell.elevation:
		// walking straight
		speed = w.WalkSpeed
	case currentCell.elevation > nextCell.elevation:
		// walking down
		speed = w.WalkDownSpeed
	case currentCell.elevation < nextCell.elevation:
		// walking up
		speed = w.WalkUpSpeed
	}

	// advance
	nextPos := src.AddCoord(away.Mult(speed))
	if src.Distance(nextPos) < src.Distance(dest) {
		return nextPos
	} else {
		return dest
	}
}

type Possesser struct {
	Items []Item
	Limit int
}

type Chaser struct {
	target int
}

func (c *Chaser) LockOn(target int) {
	c.target = target
}

func (c *Chaser) Target() int {
	return c.target
}

type Gunner struct{
	fireRange float32
	gunDamage float32
}

func (g Gunner) CanShoot(src, dest UnitCoord) bool {
	return src.Distance(dest) < g.fireRange
}

type Biter struct {
	biteDamage float32
}

func (b Biter) CanBite(src, dest UnitCoord) bool {
	return src.Distance(dest) < 1
}

type Soldier struct {
	Walker
	Possesser
	Chaser
	Gunner
	field *Field
	id    int
	health float32
}

func (s *Soldier) SetID(id int) {
	s.id = id
}

func (s *Soldier) GetID() int {
	return s.id
}

func (s *Soldier) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := s.Walker.MoveToward(s.field, src, dest)
	return s.field.MoveMe(s.id, nextCoord)
}

func (s *Soldier) RecieveDamage(from int, dmg float32) {
	s.health -= dmg
	fmt.Println("[squad] FUCK i got hit and have", s.health, "health")
	if s.health < 0 {
		fmt.Println("[squad] im dead")
		s.field.KillMe(s.id)
	}
	s.health -= dmg
}

func (s *Soldier) Shoot(src, dest UnitCoord, victim Unit) {
	// TODO: calculate hit probability based on distance
	victim.RecieveDamage(s.id, s.Gunner.gunDamage)
}

type Zed struct {
	field *Field
	id int

	Walker
	Chaser
	Biter
	health float32
	lastAttacker int

	rage float32
	nutrition float32
}

func NewZed(field *Field) *Zed {
	return &Zed{Walker: Walker{ZED_MOVER_WALK, ZED_MOVER_WALKUP, ZED_MOVER_WALKDOWN},
		Biter: Biter{biteDamage: ZED_BITE_DAMAGE}, lastAttacker: -1, rage: 0,
		nutrition: ZED_NUTRITION_BASE, health: ZED_HEALTH, field: field}
}

func (z *Zed) SetID(id int) {
	z.id = id
}

func (z *Zed) GetID() int {
	return z.id
}
func (z *Zed) MoveToward(src, dest UnitCoord) UnitCoord {
	// apply nutrition and rage speedup/slowdown
	nutr_coeff := z.nutrition / 1000
	rage_coeff := z.rage * ZED_RAGE_SPEEDUP
	all_coeff := nutr_coeff + rage_coeff
	z.Walker = Walker{fbound(ZED_MOVER_WALK * all_coeff, 0, 1),
		fbound(ZED_MOVER_WALKUP * all_coeff, 0, 1),
		fbound(ZED_MOVER_WALKDOWN * all_coeff, 0, 1)}
	nextCoord := z.Walker.MoveToward(z.field, src, dest)
	z.nutrition -= src.Distance(nextCoord) * ZED_NUTRITION_WALKING
	return z.field.MoveMe(z.id, nextCoord)
}

func (z *Zed) Bite(src, dest UnitCoord, victim Unit) {
	damage := z.Biter.biteDamage + z.rage * ZED_RAGE_BITEUP
	z.nutrition -= damage * ZED_NUTRITION_BITING

	victim.RecieveDamage(z.id, z.Biter.biteDamage)
}

func (z *Zed) RecieveDamage(from int, dmg float32) {
	z.health -= dmg
	z.rage += dmg * ZED_RAGE_FROM_DAMAGE
	z.lastAttacker = from
	fmt.Println("[zed] ARRRGH i got hit and have", z.health, "health")
	if z.health < 0 {
		fmt.Println("[zed] im finally dead")
		z.field.KillMe(z.id)
	}
}

func (z *Zed) Eat(food float32) {
	z.nutrition += food
}

func (z *Zed) Digest() bool {
	// calm down
	z.rage -= z.rage * ZED_RAGE_COOLING
	if z.rage < ZED_RAGE_THRESHOLD {
		z.rage = 0
	}

	// feed the anger
	z.nutrition -= z.rage * ZED_RAGE_COST

	// digest the food
	if z.nutrition > ZED_NUTRITION_TO_HP_THRESHOLD {
		z.nutrition -= ZED_NUTRITION_TO_HP_PORTION
		z.health += ZED_NUTRITION_TO_HP_PORTION * ZED_NUTRITION_TO_HP_SCALE
	}

	if z.nutrition < 0 {
		// starve to death
		fmt.Println("[zed] im starved to death")
		z.field.KillMe(z.id)
		return false
	}
	return true
}

type Damsel struct {
	Walker
	field *Field
	id int
	health float32
	lastAttacker int
	wanderTarget UnitCoord
}

func (d *Damsel) SetID(id int) {
	d.id = id
}

func (d *Damsel) GetID() int {
	return d.id
}

func (d *Damsel) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := d.Walker.MoveToward(d.field, src, dest)
	return d.field.MoveMe(d.id, nextCoord)
}

func (d *Damsel) MoveAway(src, dest UnitCoord) UnitCoord {
	nextCoord := d.Walker.MoveAway(d.field, src, dest)
	return d.field.MoveMe(d.id, nextCoord)
}

func (d *Damsel) RecieveDamage(from int, dmg float32) {
	d.health -= dmg
	d.lastAttacker = from
	fmt.Println("[dam] :( :( i got hit and have", d.health, "health")
	if d.health < 0 {
		fmt.Println("[dam] im dead :'(")
		d.field.KillMe(d.id)
	}
}

// temporary implement corpse as unit
type Corpse struct {
	field *Field
	id int
	unit Unit
	ressurectCounter int
}

func (c *Corpse) SetID(id int) {
	c.id = id
}

func (c *Corpse) GetID() int {
	return c.id
}

func (c *Corpse) MoveToward(src, dest UnitCoord) UnitCoord {
	return src
}

func (c *Corpse) RecieveDamage(from int, dmg float32) {}

func (c *Corpse) Respawn() *Zed {
	return NewZed(c.field)
}
