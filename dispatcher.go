package main

import (
	"log"
)

const (
	DISP_ATTACH = iota
	DISP_DETACH
)

var (
	singlePlayerRules = Rules{minPlayers: 1, maxPlayers: 1}
)

type Dispatcher struct {
	field       *Field
	players     []Player
	rules       Rules
	lastid      int
	playerQueue chan PlayerReq
	time        *Time
	gameState   chan GameState
}

type Player struct {
	render Render
	Id     int
	Orders chan Order
}

type PlayerReq struct {
	p    Player
	op   int
	Id   int
	resp chan int
}

type Rules struct {
	minPlayers int
	maxPlayers int
	canIdle    bool
}

func NewDispatcher(r Rules) *Dispatcher {
	return &Dispatcher{rules: r, playerQueue: make(chan PlayerReq),
		time: NewTime(TIME_TICKS_PER_SEC)}
}

func (d *Dispatcher) AttachPlayer(r Render) int {
	req := PlayerReq{Player{render: r}, DISP_ATTACH, -1, make(chan int, 1)}
	d.playerQueue <- req
	return <-req.resp
}

func (d *Dispatcher) DetachPlayer(Id int) {
	req := PlayerReq{Player{}, DISP_ATTACH, Id, make(chan int, 1)}
	d.playerQueue <- req
	<-req.resp
}

func (d *Dispatcher) handlePlayerReq(r PlayerReq) {
	switch r.op {
	case DISP_ATTACH:
		d.players = append(d.players, r.p)
		Pid := d.lastid
		d.lastid++
		d.players[len(d.players)-1].Id = Pid
		r.resp <- Pid
		log.Println("dp: attached player, id", Pid)
	case DISP_DETACH:
		for idx, p := range d.players {
			if p.Id == r.Id {
				// TODO: close render?
				copy(d.players[:idx], d.players[idx+1:])
				d.players = d.players[:len(d.players)-1]
			}
		}
		r.resp <- 0
	}
}

func (d *Dispatcher) countPlayers() int {
	return len(d.players)
}

func (d *Dispatcher) playerById(Id int) *Player {
	for idx := range d.players {
		if d.players[idx].Id == Id {
			return &d.players[idx]
		}
	}
	return nil
}

func (d *Dispatcher) Run() {
	for {
		// generate field
		d.field = generateField()
		d.gameState = d.field.gameState

		// wait for desired amount of players to join
		// TODO: allow abort
		for {
			req := <-d.playerQueue
			d.handlePlayerReq(req)
			if d.countPlayers() >= d.rules.minPlayers {
				log.Println("dp: enough players")
				break
			}
		}
		// run game
		d.runGame()
		// cleanup
	}
}

func (d *Dispatcher) runGame() {
	// bind players to squads
	for idx, player := range d.players {
		player.Orders = placeSquad(d.field, idx, player.Id)
		player.render.AssignSquad(player.Id, player.Orders)
	}

	populateField(d.field)

	// start game timer
	d.time.SetTicker(d.field)
	go d.time.Run()
	for {
		select {
		case field := <-d.field.updates:
			// sanitize field and send it to all players
			for _, p := range d.players {
				p.render.HandleUpdate(field)
			}
		case pr := <-d.playerQueue:
			d.handlePlayerReq(pr)
			if pr.op == DISP_ATTACH {
				d.players[len(d.players)-1].render.Spectate()
			}
		case state := <-d.gameState:
			if state.player >= 0 {
				player := d.playerById(state.player)
				if player == nil {
					continue
				}
				player.render.HandleGameState(state)
			} else {
				for _, p := range d.players {
					p.render.HandleGameState(state)
				}
			}
			// TODO: abort channel
		}
	}
}
