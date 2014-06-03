package main

import (
	"encoding/gob"
	"log"
	"net"
)

type RemoteRender struct {
	updates      chan *Field
	Orders       chan Order
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

func (rr *RemoteRender) AssignSquad(Id int, Orders chan Order) {
	rr.assignments <- Assignment{Id, Orders}
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
	var order Order
	for {
		err := decoder.Decode(&order)
		if err != nil {
			rr.readErrs <- err
			return
		}
		log.Println("rr: recieved order")

		select {
		case rr.Orders <- order:
		default:
		}
	}
}

func (rr *RemoteRender) runWriter() {
	encoder := gob.NewEncoder(rr.conn)
	for {
		select {
		case Assignment := <-rr.assignments:
			log.Println("rr: sending assign")
			err := encoder.Encode(UpdateBulk{Assignment: &Assignment})
			if err != nil {
				rr.writeErrs <- err
				return
			}
			rr.Orders = Assignment.Orders
		case field := <-rr.localUpdates:
			//log.Println("rr: sending field", len(field.Cells))
			err := encoder.Encode(UpdateBulk{Field: field})
			//log.Println("rr: sent field")
			if err != nil {
				rr.writeErrs <- err
				return
			}
		case State := <-rr.localStateUpdates:
			log.Println("rr: sending state")
			err := encoder.Encode(UpdateBulk{GameState: &State})
			if err != nil {
				rr.writeErrs <- err
				return
			}
		}
	}
}
