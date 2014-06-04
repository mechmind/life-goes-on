package main

import (
	"encoding/gob"
)

func init() {
	gob.Register(&Field{})

	gob.Register(&Zed{})
	gob.Register(&Damsel{})
	gob.Register(&Soldier{})
	gob.Register(&Corpse{})

	gob.Register(&ZedSwarm{})
	gob.Register(&DamselCrowd{})
	gob.Register(&Squad{})
	gob.Register(&NopAgent{})

	gob.Register(GameState{})
	gob.Register(Assignment{})
	gob.Register(Order{})
}

type UpdateBulk struct {
	Field      *Field
	Assignment *Assignment
	GameState  *GameState
	Reset      bool
	Message    *Message
}
