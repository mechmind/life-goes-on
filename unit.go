package main

import (
	"fmt"
)

type Unit interface {
	SetID(int)
	Mover
}

type Mover interface {
	MoveToward(src, dest UnitCoord) UnitCoord
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

	fmt.Printf("[unit] moving from %s toward %s: toward=%s, next cell=%s\n", src, dest,
		toward, nextCellCoord)
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

type Soldier struct {
	Walker
	Possesser
	field *Field
	id    int
}

func (s *Soldier) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := s.Walker.MoveToward(s.field, src, dest)
	return s.field.MoveMe(s.id, nextCoord)
}

func (s *Soldier) SetID(id int) {
	s.id = id
}

type Zed struct {
	Walker
	field *Field
	id int
}

func (z *Zed) MoveToward(src, dest UnitCoord) UnitCoord {
	nextCoord := z.Walker.MoveToward(z.field, src, dest)
	return z.field.MoveMe(z.id, nextCoord)
}

func (z *Zed) SetID(id int) {
	z.id = id
}
