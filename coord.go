package main

import (
	"fmt"
	"math"
)

type UnitCoord struct {
	X, Y float32
}

func (u UnitCoord) String() string {
	return fmt.Sprintf("(%.2f, %.2f)", u.X, u.Y)
}

func (u UnitCoord) Add(x, y float32) UnitCoord {
	return UnitCoord{u.X + x, u.Y + y}
}

func (u UnitCoord) AddCoord(add UnitCoord) UnitCoord {
	return UnitCoord{u.X + add.X, u.Y + add.Y}
}

func (u UnitCoord) Mult(value float32) UnitCoord {
	return UnitCoord{u.X * value, u.Y * value}
}

func (u UnitCoord) Distance(to UnitCoord) float32 {
	dx := u.X - to.X
	dy := u.Y - to.Y
	if fabs(dx) < FLOAT_ERROR && fabs(dy) < FLOAT_ERROR {
		return 0
	}
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

func (u UnitCoord) Cell() CellCoord {
	return CellCoord{int(u.X), int(u.Y)}
}

func NormTowardCoord(src, dest UnitCoord) UnitCoord {
	toward := UnitCoord{dest.X - src.X, dest.Y - src.Y}

	cx := toward.X
	cy := toward.Y
	norm := float32(math.Sqrt(float64(cx*cx + cy*cy)))
	normToward := UnitCoord{float32(math.Sqrt(float64(cx * cx))),
		float32(math.Sqrt(float64(cy * cy)))}

	if cx < 0 {
		normToward.X = -normToward.X
	}
	if cy < 0 {
		normToward.Y = -normToward.Y
	}

	if norm < FLOAT_ERROR {
		return UnitCoord{0, 0}
	} else {
		return normToward.Mult(1.0 / norm)
	}
}

func NextCellCoord(pos, dir UnitCoord) CellCoord {
	joint := pos.Cell()
	if dir.X > 0 {
		joint.X++
	}
	if dir.Y > 0 {
		joint.Y++
	}
	jointDir := joint.Unit().AddCoord(dir.Mult(-1))
	crossProd := dir.X*jointDir.Y - dir.Y*jointDir.X
	if crossProd < 0 {
		// dir vector points to the cell counterclockwise to joint
		switch {
		case dir.X < 0 && dir.Y < 0:
			return joint.Add(0, -1)
		case dir.X < 0 && dir.Y >= 0:
			return joint.Add(-1, 0)
		case dir.X >= 0 && dir.Y >= 0:
			return joint.Add(0, 1)
		case dir.X >= 0 && dir.Y < 0:
			return joint.Add(1, 0)
		}
	} else {
		// dir vector points to the cell clockwise to joint
		switch {
		case dir.X < 0 && dir.Y < 0:
			return joint.Add(-1, 0)
		case dir.X < 0 && dir.Y >= 0:
			return joint.Add(0, 1)
		case dir.X >= 0 && dir.Y >= 0:
			return joint.Add(1, 0)
		case dir.X >= 0 && dir.Y < 0:
			return joint.Add(0, -1)
		}
	}
	return joint
}

type CellCoord struct {
	X, Y int
}

func (c CellCoord) String() string {
	return fmt.Sprintf("[%d, %d]", c.X, c.Y)
}

func (c CellCoord) Add(x, y int) CellCoord {
	return CellCoord{c.X + x, c.Y + y}
}

func (c CellCoord) AddCoord(add CellCoord) CellCoord {
	return CellCoord{c.X + add.X, c.Y + add.Y}
}

func (c CellCoord) Mul(mul int) CellCoord {
	return CellCoord{c.X * mul, c.Y * mul}
}

func (c CellCoord) Unit() UnitCoord {
	return UnitCoord{float32(c.X), float32(c.Y)}
}

func (c CellCoord) UnitCenter() UnitCoord {
	return UnitCoord{float32(c.X) + 0.5, float32(c.Y) + 0.5}
}

func CheckCellCoordBounds(value, low, high CellCoord) bool {
	if low.X <= value.X && value.X <= high.X &&
		low.Y <= value.Y && value.Y <= high.Y {

		return true
	}
	return false
}
