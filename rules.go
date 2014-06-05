package main

import (
	"errors"
	"fmt"
)

var (
	allRules = map[string]Rules{
		"single": Rules{minPlayers: 1, maxPlayers: 1},
		"duel": Rules{minPlayers: 2, maxPlayers: 2, versus: true},
		"classic": Rules{minPlayers: 2, maxPlayers: 4},
		"king-of-hill": Rules{minPlayers: 2, maxPlayers: 4, versus: true},
		"survival": Rules{minPlayers: 2, maxPlayers: 4, moreBs: 100, moreBsP: 40},
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

	var aggr string
	if r.versus {
		aggr = "versus"
	} else {
		aggr = "coop"
	}

	var moreBBs string
	if r.moreBs > 0 {
		moreBBs = fmt.Sprintf("+100 base Bs")
	}

	var morePBs string
	if r.moreBsP > 0 {
		morePBs = fmt.Sprintf("+100 perP Bs")
	}

	rules := []string{r.name, players, aggr, moreBBs, morePBs}
	return "rules: " + joinNonEmptyStrings(rules, ", ")
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

