package main

import (
	"errors"
	"fmt"
)

var (
	allRules = map[string]Rules{
		"single": Rules{minPlayers: 1, maxPlayers: 1},
		"bandits": Rules{minPlayers: 2, maxPlayers: 2, versus: true},
		"classic": Rules{minPlayers: 2, maxPlayers: 4},
		"king-of-hill": Rules{minPlayers: 2, maxPlayers: 4, versus: true},
		"crowds": Rules{minPlayers: 2, maxPlayers: 4, moreBs: 100, moreBsP: 40},
	}
)

type Rules struct {
	minPlayers int
	maxPlayers int
	versus bool
	moreBs int
	moreBsP int

	name string
}

func (r Rules) String() string {
	var info = []string{r.name}
	var players string
	if r.minPlayers == r.maxPlayers {
		if r.minPlayers == 1 {
			players = fmt.Sprintf("%d player", r.minPlayers)
		} else {
			players = fmt.Sprintf("%d players", r.minPlayers)
		}
	} else {
		players = fmt.Sprintf("%d..%d players", r.minPlayers, r.maxPlayers)
	}
	info = append(info, players)

	if r.versus {
		info = append(info, "versus")
	} else {
		info = append(info, "coop")
	}

	if r.moreBs > 0 {
		info = append(info, fmt.Sprintf("+%d base Bs", r.moreBs))
	}

	if r.moreBsP > 0 {
		info = append(info, fmt.Sprintf("+%d perP Bs", r.moreBsP))
	}

	return "rules: " + joinNonEmptyStrings(info, ", ")
}

type Ruleset []Rules

func (r *Ruleset) AddRules(name string) error {
	rule, ok := allRules[name]
	if ! ok {
		return errors.New("no such rule: " + name)
	}

	rule.name = name
	// TODO: check for incompatible rules
	*r = append(*r, rule)
	return nil
}

