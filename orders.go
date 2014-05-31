package main

const (
	ORDER_MOVE = iota
	ORDER_FIRE
	ORDER_NOFIRE
	ORDER_SEMIFIRE
	ORDER_GREN
)

type Order struct {
	order int
	coord CellCoord
}
