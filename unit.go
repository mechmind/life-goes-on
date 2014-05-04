package main

import (
	"fmt"
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
	Walker
	Chaser
	Biter
	field *Field
	id int
	health float32
	lastAttacker int
}

func (z *Zed) SetID(id int) {
	z.id = id
}

func (z *Zed) GetID() int {
	return z.id
}
func (z *Zed) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := z.Walker.MoveToward(z.field, src, dest)
	return z.field.MoveMe(z.id, nextCoord)
}

func (z *Zed) Bite(src, dest UnitCoord, victim Unit) {
	victim.RecieveDamage(z.id, z.Biter.biteDamage)
}

func (z *Zed) RecieveDamage(from int, dmg float32) {
	z.health -= dmg
	z.lastAttacker = from
	fmt.Println("[zed] ARRRGH i got hit and have", z.health, "health")
	if z.health < 0 {
		fmt.Println("[zed] im finally dead")
		z.field.KillMe(z.id)
	}
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
