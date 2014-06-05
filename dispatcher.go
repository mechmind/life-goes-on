package main

import (
	"fmt"
	"log"
	"time"
)

const (
	DISP_ATTACH = iota
	DISP_DETACH
)

const (
	GAMEOVER_COUNTDOWN = 5
)


type Dispatcher struct {
	field       *Field
	players     []Player
	rules       *Ruleset
	currentRules int
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


func NewDispatcher(r *Ruleset) *Dispatcher {
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
		d.sendAll(MESSAGE_LEVEL_INFO, "player joined the match")
		log.Println("dispatcher: attached new player with id", Pid)
	case DISP_DETACH:
		defer func() { r.resp <- 0 }()
		for idx, p := range d.players {
			if p.Id == r.Id {
				if p.Orders != nil {
					// kill squad
					p.Orders <- Order{ORDER_SUICIDE, CellCoord{}}
				}
				copy(d.players[idx:], d.players[idx+1:])
				d.players = d.players[:len(d.players)-1]
				d.sendAll(MESSAGE_LEVEL_INFO, "player left the match")
				log.Println("dispatcher: detached player with id", p.Id)
				return
			}
		}
		log.Printf("dispatcher: cannot detach player: no player with id", r.Id)
	}
	log.Println("dispatcher: total players now:", d.countPlayers())
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
	log.Println("dispatcher: starting up")
	for {
		rules := (*d.rules)[d.currentRules]

		log.Printf("dispatcher: starting new round with rule '%s', generating field", rules.name)
		// generate field
		d.field = generateField(rules)
		d.gameState = d.field.gameState

		// reset state of existing players
		log.Println("dispatcher: resetting state for all connected players")
		for _, p := range d.players {
			p.render.Reset()
			p.render.Spectate()
			p.render.HandleUpdate(d.field)
			p.render.HandleGameState(GameState{GAME_WAIT, -1})
		}

		// wait for desired amount of players to join
		log.Println("dispatcher: waiting for players")
		for {
			if d.countPlayers() >= rules.minPlayers {
				break
			}
			d.sendAll(MESSAGE_LEVEL_INFO,
				fmt.Sprintf("waiting for %d players to join...",
					rules.minPlayers-d.countPlayers()))
			req := <-d.playerQueue
			d.handlePlayerReq(req)
			if req.op == DISP_ATTACH {
				newPlayer := d.players[len(d.players)-1]
				log.Println("dispatcher: new player in wait stage, set it up")
				newPlayer.render.Spectate()
				newPlayer.render.HandleUpdate(d.field)
			} else {
				log.Println("dispatcher: player %d detached in wait stage", req.Id)
			}
		}
		// run game
		d.runGame()
		// cleanup
		d.currentRules = (d.currentRules + 1) % len(*d.rules)
	}
}

func (d *Dispatcher) runGame() {
	// bind players to squads
	log.Println("dispatcher: starting game")
	rules := (*d.rules)[d.currentRules]
	for idx, Player := range d.players {
		Player.render.HandleGameState(GameState{GAME_RUNNING, -1})
		if idx < rules.maxPlayers {
			log.Printf("dispatcher: player %d now control squad", Player.Id)
			Player.Orders = placeSquad(d.field, idx, Player.Id, rules)
			Player.render.AssignSquad(Player.Id, Player.Orders)
			d.players[idx].Orders = Player.Orders
		} else {
			log.Printf("dispatcher: player %d spectating", Player.Id)
			Player.render.Spectate()
		}
	}

	log.Println("dispatcher: populating field")
	populateField(d.field, rules)

	// start game timer
	d.time.SetTicker(d.field)
	var countdownTicker <-chan time.Time
	go d.time.Run()
	defer d.time.Stop()

	d.sendAll(MESSAGE_LEVEL_INFO, rules.String())

	var countdownMsg = "new round in "
	var countdown = GAMEOVER_COUNTDOWN
	log.Println("dispatcher: entering game")
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
				log.Println("dispatcher: new player in middle of round, spectate")
				d.players[len(d.players)-1].render.Spectate()
			} else {
				log.Printf("dispatcher: player %d have quit in middle of round", pr.Id)
			}
		case State := <-d.gameState:
			if State.Player >= 0 {
				Player := d.playerById(State.Player)
				if Player == nil {
					continue
				}
				Player.render.HandleGameState(State)
				switch {
				case State.State & GAME_LOSE > 0:
					d.sendAll(MESSAGE_LEVEL_INFO,
						fmt.Sprintf("player %d have been exterminated", State.Player))
				case State.State & GAME_WIN > 0:
					d.sendAll(MESSAGE_LEVEL_INFO, fmt.Sprintf("player %d have won!", State.Player))
				}
			} else {
				for _, p := range d.players {
					p.render.HandleGameState(State)
				}
			}

			if State.State == GAME_OVER {
				// game is over
				log.Println("dispatcher: game is over")
				countdownTicker = time.Tick(time.Second)
			}
		case <-countdownTicker:
			countdown--
			countdownMsg += fmt.Sprintf("%d... ", countdown)
			d.sendAll(MESSAGE_LEVEL_INFO, countdownMsg)
			if countdown == 0 {
				return
			}
		}
	}
}

func (d *Dispatcher) sendAll(lvl int, msg string) {
	for _, p := range d.players {
		p.render.HandleMessage(lvl, msg)
	}
}
