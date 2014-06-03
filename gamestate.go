package main

const (
	GAME_WAIT = 1 << iota
	GAME_RUNNING
	GAME_WIN
	GAME_LOSE
	GAME_DRAW
	GAME_OVER
)

type GameState struct {
	state  int
	player int
}
