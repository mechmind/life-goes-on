package main

import (
	"encoding/gob"
	"net"
)

type RemoteRender struct {
	updates      chan *Field
	orders       chan Order
	squad        int
	stateUpdates chan GameState
	assignments  chan Assignment

	localUpdates        chan *Field
	localStateUpdates   chan GameState
	conn                *net.TCPConn
	readErrs, writeErrs chan error
	mapSent             bool
}

func CreateRemoteRender(conn *net.TCPConn) *RemoteRender {
	return &RemoteRender{updates: make(chan *Field, 3), stateUpdates: make(chan GameState, 3),
		squad: -1, assignments: make(chan Assignment, 1),
		localUpdates: make(chan *Field, 3), localStateUpdates: make(chan GameState, 3),
		conn: conn, readErrs: make(chan error), writeErrs: make(chan error)}
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

func (rr *RemoteRender) AssignSquad(id int, orders chan Order) {
	rr.assignments <- Assignment{id, orders}
}

func (rr *RemoteRender) Spectate() {
	rr.assignments <- Assignment{-1, nil}
}

func (rr *RemoteRender) Reset() {
	rr.mapSent = false
}

func (rr *RemoteRender) Run() error {
	go rr.runReader()
	go rr.runWriter()
	for {
		select {
		// local channels
		//case assignment := <-rr.assignments: // handled directly by writer
		case field := <-rr.updates:
			if rr.mapSent {
				field.cells = nil
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
	var order Order
	for {
		err := decoder.Decode(&order)
		if err != nil {
			rr.readErrs <- err
			return
		}

		select {
		case rr.orders <- order:
		default:
		}
	}
}

func (rr *RemoteRender) runWriter() {
	encoder := gob.NewEncoder(rr.conn)
	for {
		select {
		case Assignment := <-rr.assignments:
			err := encoder.Encode(Assignment)
			if err != nil {
				rr.writeErrs <- err
				return
			}
		case field := <-rr.localUpdates:
			err := encoder.Encode(field)
			if err != nil {
				rr.writeErrs <- err
				return
			}
		case state := <-rr.localStateUpdates:
			err := encoder.Encode(state)
			if err != nil {
				rr.writeErrs <- err
				return
			}
		}
	}
}
