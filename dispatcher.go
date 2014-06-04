package main

import (
	"log"
	"time"
)

const (
	DISP_ATTACH = iota
	DISP_DETACH
)

const (
	GAMEOVER_FADEOUT_TIME = time.Second * 5
)

var (
	singlePlayerRules = Rules{minPlayers: 1, maxPlayers: 1}
	duelRules         = Rules{minPlayers: 2, maxPlayers: 2}
	coopRules         = Rules{minPlayers: 1, maxPlayers: 2}
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
	req := PlayerReq{Player{}, DISP_DETACH, Id, make(chan int, 1)}
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
				log.Println("dp: removing player", p.Id)
				if p.Orders != nil {
					// kill squad
					log.Println("dp: killing orphan squad of", p.Id)
					p.Orders <-Order{ORDER_SUICIDE, CellCoord{}}
				}
				copy(d.players[:idx], d.players[idx+1:])
				d.players = d.players[:len(d.players)-1]
				break
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

		// reset state of existing players
		for _, p := range d.players {
			p.render.Reset()
			p.render.Spectate()
			p.render.HandleUpdate(d.field)
			p.render.HandleGameState(GameState{GAME_WAIT, -1})
		}

		// wait for desired amount of players to join
		for {
			if d.countPlayers() >= d.rules.minPlayers {
				break
			}
			req := <-d.playerQueue
			d.handlePlayerReq(req)
			if req.op == DISP_ATTACH {
				newPlayer := d.players[len(d.players)-1]
				newPlayer.render.Spectate()
				newPlayer.render.HandleUpdate(d.field)
			}
		}
		// run game
		d.runGame()
		// cleanup
	}
}

func (d *Dispatcher) runGame() {
	// bind players to squads
	for idx, Player := range d.players {
		Player.render.HandleGameState(GameState{GAME_RUNNING, -1})
		if idx < d.rules.maxPlayers {
			Player.Orders = placeSquad(d.field, idx, Player.Id)
			Player.render.AssignSquad(Player.Id, Player.Orders)
			d.players[idx].Orders = Player.Orders
		} else {
			Player.render.Spectate()
		}
	}

	populateField(d.field)

	// start game timer
	d.time.SetTicker(d.field)
	var fadeout <-chan time.Time
	go d.time.Run()
	defer d.time.Stop()
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
			} else {
				log.Println("disp: detached player", pr.Id)
			}
		case State := <-d.gameState:
			if State.Player >= 0 {
				log.Println("disp: got news for", State.Player, "news is", State.State)
				Player := d.playerById(State.Player)
				if Player == nil {
					continue
				}
				Player.render.HandleGameState(State)
			} else {
				log.Println("disp: got news for everyone news is", State.State)
				for _, p := range d.players {
					p.render.HandleGameState(State)
				}
			}

			if State.State == GAME_OVER {
				// game is over
				fadeout = time.After(GAMEOVER_FADEOUT_TIME)
			}
		case <-fadeout:
			return
		}
	}
}
