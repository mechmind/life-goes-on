package main

import (
	"fmt"
)

const (
	GAME_WAIT = 1 << iota
	GAME_RUNNING
	GAME_OVER

	GAME_WIN
	GAME_LOSE
	GAME_DRAW
)

type GameState struct {
	State  int
	Player int
}

func (g GameState) String() string {
	var str string
	var state = g.State
	switch {
	case state&GAME_WAIT > 0:
		str = "WAIT"
	case state&GAME_RUNNING > 0:
		str = "RUNNING"
	case state&GAME_OVER > 0:
		str = "OVER"
	}

	var end string
	switch {
	case state&GAME_WIN > 0:
		end = "WIN"
	case state&GAME_LOSE > 0:
		end = "LOSE"
	case state&GAME_DRAW > 0:
		end = "DRAW"
	}
	if end != "" {
		if str != "" {
			str += " " + end
		} else {
			str = end
		}
	}

	if g.Player >= 0 {
		return fmt.Sprintf("state{%s, player %d}", str, g.Player)
	} else {
		return fmt.Sprintf("state{%s, all players}", str)
	}
}
