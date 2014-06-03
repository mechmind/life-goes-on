package main

import (
	"encoding/gob"
)

func init() {
	gob.Register(&Field{})

	gob.Register(Zed{})
	gob.Register(Damsel{})
	gob.Register(Soldier{})

	gob.Register(ZedSwarm{})
	gob.Register(DamselCrowd{})
	gob.Register(Squad{})

	gob.Register(GameState{})
	gob.Register(Assignment{})
	gob.Register(Order{})
	gob.Register()
}
