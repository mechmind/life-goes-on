package main

const (
	OBJECT_EMPTY = iota
	OBJECT_WALL
	OBJECT_BUSH
	OBJECT_BARRICADE
)

var referenceObjects = []Object {
	OBJECT_EMPTY: {Type: OBJECT_EMPTY, Passable: true},
	OBJECT_WALL: {Type: OBJECT_WALL, Passable: false, Opaque: true, Health: -1},
	OBJECT_BUSH: {Type: OBJECT_BUSH, Passable: true, Opaque: true, Health: 220},
	OBJECT_BARRICADE: {Type: OBJECT_BARRICADE, Passable: false, Opaque: false, Health: 720},
}

type Object struct {
	Type int
	Health float32
	Passable bool
	Opaque bool
}


