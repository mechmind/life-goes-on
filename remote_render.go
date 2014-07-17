package main

import (
	"encoding/gob"
	"log"
	"net"
)

const (
	REMOTE_ENCODE_BUFFER_SIZE = 8 * 1024 * 1024
)

type RemoteRender struct {
	updates      chan *Field
	Orders       chan Order
	messages     chan Message
	squad        int
	stateUpdates chan GameState
	assignments  chan Assignment

	localUpdates        chan *Field
	localStateUpdates   chan GameState
	conn                *net.TCPConn
	readErrs, writeErrs chan error
	mapSent             bool
	reset               chan chan struct{}
}

func CreateRemoteRender(conn *net.TCPConn) *RemoteRender {
	return &RemoteRender{updates: make(chan *Field, 3), stateUpdates: make(chan GameState, 3),
		squad: -1, assignments: make(chan Assignment, 1),
		localUpdates: make(chan *Field, 3), localStateUpdates: make(chan GameState, 3),
		conn: conn, readErrs: make(chan error), writeErrs: make(chan error),
		reset: make(chan chan struct{}, 1), messages: make(chan Message, 3)}
}

func (rr *RemoteRender) HandleUpdate(f *Field) {
	select {
	case rr.updates <- f:
	default:
	}
}

func (rr *RemoteRender) HandleGameState(s GameState) {
	select {
	case rr.stateUpdates <- s:
	default:
	}
}

func (rr *RemoteRender) HandleMessage(lvl int, msg string) {
	select {
	case rr.messages <- Message{lvl, msg, MESSAGE_TTL}:
	default:
	}
}

func (rr *RemoteRender) AssignSquad(Id int, Orders chan Order) {
	rr.assignments <- Assignment{Id, Orders}
}

func (rr *RemoteRender) Spectate() {
	rr.assignments <- Assignment{-1, nil}
}

func (rr *RemoteRender) Reset() {
	rr.mapSent = false
	confirm := make(chan struct{})
	rr.reset <- confirm
	<-confirm
}

func (rr *RemoteRender) Run() error {
	log.Println("Rrender: starting up")
	go rr.runReader()
	go rr.runWriter()
	for {
		select {
		// local channels
		//case assignment := <-rr.assignments: // handled directly by writer
		//case <-rr.reset // handled directly by writer
		case field := <-rr.updates:
			field = copyField(field)
			if rr.mapSent {
				field.Cells = nil
			} else {
				rr.mapSent = true
			}

			rr.localUpdates <- field
		case gameState := <-rr.stateUpdates:
			// TODO: handle somewhat
			rr.localStateUpdates <- gameState

		// orders sent directly from reader
		// error channels
		case err := <-rr.readErrs:
			return err
		case err := <-rr.writeErrs:
			return err
		}
	}
	return nil
}

func (rr *RemoteRender) runReader() {
	// read remote data
	decoder := gob.NewDecoder(rr.conn)
	for {
		var Order Order
		err := decoder.Decode(&Order)
		if err != nil {
			rr.readErrs <- err
			return
		}

		select {
		case rr.Orders <- Order:
		default:
		}
	}
}

func (rr *RemoteRender) runWriter() {
	encoder := gob.NewEncoder(rr.conn)
	for {
		select {
		case Assignment := <-rr.assignments:
			err := encoder.Encode(UpdateBulk{Assignment: &Assignment})
			if err != nil {
				rr.writeErrs <- err
				return
			}
			rr.Orders = Assignment.Orders
		case field := <-rr.localUpdates:
			err := encoder.Encode(UpdateBulk{Field: field})
			if err != nil {
				rr.writeErrs <- err
				return
			}
		case State := <-rr.localStateUpdates:
			err := encoder.Encode(UpdateBulk{GameState: &State})
			if err != nil {
				rr.writeErrs <- err
				return
			}
		case msg := <-rr.messages:
			err := encoder.Encode(UpdateBulk{Message: &msg})
			if err != nil {
				rr.writeErrs <- err
				return
			}

		case confirm := <-rr.reset:
			err := encoder.Encode(UpdateBulk{Reset: true})
			confirm <- struct{}{}
			if err != nil {
				rr.writeErrs <- err
				return
			}
		}
	}
}
