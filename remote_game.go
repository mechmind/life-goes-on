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
	orders              chan Order
	readErrs, writeErrs chan error
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
		orders: make(chan Order)}
	return rg, nil
}

func (rg *RemoteGame) AttachPlayer(r Render) {
	rg.render = r
}

func (rg *RemoteGame) Run() {
	go rg.runReader()
	go rg.runWriter()

	defer rg.conn.Close()
	select {
	case err := <-rg.readErrs:
		log.Println("conn: read failed:", err)
	case err := <-rg.writeErrs:
		log.Println("conn: write failed:", err)
	}
}

func (rg *RemoteGame) runReader() {
	decoder := gob.NewDecoder(rg.conn)
	var i interface{}
	for {
		err := decoder.Decode(&i)
		if err != nil {
			rg.readErrs <- err
			return
		}

		switch i.(type) {
		case *Field:
			// is an update
			field := i.(*Field)
			rg.render.HandleUpdate(field)
		case Assignment:
			// is an assignment
			ass := i.(Assignment)
			rg.render.AssignSquad(ass.id, rg.orders)
		case GameState:
			// is an game state
			state := i.(GameState)
			rg.render.HandleGameState(state)
		default:
			rg.readErrs <- errors.New("unknown type decoded")
			return
		}
	}
}

func (rg *RemoteGame) runWriter() {
	encoder := gob.NewEncoder(rg.conn)
	for {
		order := <-rg.orders
		err := encoder.Encode(order)
		if err != nil {
			rg.writeErrs <- err
			return
		}
	}
}
