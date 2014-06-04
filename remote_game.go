package main

import (
	"encoding/gob"
	"errors"
	"log"
	"net"
)

type RemoteGame struct {
	conn *net.TCPConn

	render              Render
	Orders              chan Order
	readErrs, writeErrs chan error
	attachan            chan Render
	cells               []Cell
}

func ConnectRemoteGame(straddr string) (*RemoteGame, error) {
	// connect
	addr, err := net.ResolveTCPAddr("tcp4", straddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp4", nil, addr)
	if err != nil {
		return nil, err
	}

	// send handshake
	header := []byte{'L', 'G', 'O', PROTO_VERSION}
	_, err = conn.Write(header)
	if err != nil {
		conn.Close()
		return nil, err
	}

	rg := &RemoteGame{conn: conn, readErrs: make(chan error), writeErrs: make(chan error),
		Orders: make(chan Order), attachan: make(chan Render)}
	return rg, nil
}

func (rg *RemoteGame) AttachPlayer(r Render) int {
	rg.attachan <- r
	return 0
}

func (rg *RemoteGame) Run() {
	// wait for attaching
	log.Println("Rgame: waiting for render...")
	rg.render = <-rg.attachan
	log.Println("Rgame: render attached, starting chat with server")
	// then interact with remote
	go rg.runReader()
	go rg.runWriter()

	defer rg.conn.Close()
	select {
	case err := <-rg.readErrs:
		log.Println("Rgame: failed to recieve message from server:", err)
	case err := <-rg.writeErrs:
		log.Println("Rgame: failed to send message to server:", err)
	}
	log.Println("Rgame: error occured, giving up")
	rg.render.HandleMessage(MESSAGE_LEVEL_INFO, "connection to server have been terminated")
}

func (rg *RemoteGame) runReader() {
	decoder := gob.NewDecoder(rg.conn)
	for {
		var ub UpdateBulk
		err := decoder.Decode(&ub)
		if err != nil {
			rg.readErrs <- err
			return
		}

		switch {
		case ub.Field != nil:
			// is an update
			field := ub.Field
			//log.Println("rg: got field", len(field.Cells))
			rg.fixField(field)
			rg.render.HandleUpdate(field)
		case ub.Assignment != nil:
			// is an assignment
			ass := *ub.Assignment
			rg.render.AssignSquad(ass.Id, rg.Orders)
		case ub.GameState != nil:
			// is an game state
			State := *ub.GameState
			rg.render.HandleGameState(State)
		case ub.Message != nil:
			msg := ub.Message
			rg.render.HandleMessage(msg.Level, msg.Content)
		case ub.Reset == true:
			rg.render.Reset()
		default:
			rg.readErrs <- errors.New("all bulk fields are nil")
			return
		}
	}
}

func (rg *RemoteGame) runWriter() {
	encoder := gob.NewEncoder(rg.conn)
	for {
		Order := <-rg.Orders
		err := encoder.Encode(Order)
		if err != nil {
			rg.writeErrs <- err
			return
		}
	}
}

func (rg *RemoteGame) fixField(field *Field) {
	for idx := range field.Units {
		switch field.Units[idx].Unit.(type) {
		case *Zed:
			u := field.Units[idx].Unit.(*Zed)
			u.field = field
		case *Damsel:
			u := field.Units[idx].Unit.(*Damsel)
			u.field = field
		case *Soldier:
			u := field.Units[idx].Unit.(*Soldier)
			u.field = field
		case *Corpse:
			u := field.Units[idx].Unit.(*Corpse)
			u.field = field
		}
	}
	if field.Cells != nil {
		rg.cells = field.Cells
	} else {
		field.Cells = rg.cells
	}
}
