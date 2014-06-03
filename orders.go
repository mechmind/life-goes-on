package main

const (
	ORDER_MOVE = iota
	ORDER_AUTOMOVE
	ORDER_FIRE
	ORDER_NOFIRE
	ORDER_SEMIFIRE
	ORDER_GREN
)

type Order struct {
	order int
	Coord CellCoord
}

func toggleFireState(fs int) int {
	switch fs {
	case ORDER_FIRE:
		return ORDER_SEMIFIRE
	case ORDER_SEMIFIRE:
		return ORDER_FIRE
	default:
		return ORDER_FIRE
	}
}
