package main

import (
	"fmt"
)

type Unit interface {
	SetID(int)
	Mover
	DamageReciever
}

type Mover interface {
	MoveToward(src, dest UnitCoord) UnitCoord
}

type DamageReciever interface {
	RecieveDamage(dmg float32)
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

func (s *Soldier) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := s.Walker.MoveToward(s.field, src, dest)
	return s.field.MoveMe(s.id, nextCoord)
}

func (s *Soldier) RecieveDamage(dmg float32) {
	s.health -= dmg
}

func (s *Soldier) Shoot(src, dest UnitCoord, victim Unit) {
	// TODO: calculate hit probability based on distance
	victim.RecieveDamage(s.Gunner.gunDamage)
}

type Zed struct {
	Walker
	Chaser
	field *Field
	id int
	health float32
}

func (z *Zed) SetID(id int) {
	z.id = id
}

func (z *Zed) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := z.Walker.MoveToward(z.field, src, dest)
	return z.field.MoveMe(z.id, nextCoord)
}

func (z *Zed) RecieveDamage(dmg float32) {
	z.health -= dmg
	fmt.Println("[zed] ARRRGH i got hit and have", z.health, "health")
}
