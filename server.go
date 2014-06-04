package main

import (
	"log"
	"net"
)

const (
	PROTO_VERSION = 1
)

type Server struct {
	dispatcher *Dispatcher
	listener   *net.TCPListener
}

func CreateServer(dispatcher *Dispatcher, straddr string) (*Server, error) {
	addr, err := net.ResolveTCPAddr("tcp4", straddr)
	if err != nil {
		return nil, err
	}

	server := &Server{dispatcher: dispatcher}
	server.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (s *Server) Serve() {
	log.Println("server: accepting connections")
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			log.Println("server: failed to accept connection:", err)
		} else {
			log.Printf("server: new connection from '%s'", conn.RemoteAddr())
			go s.serveConn(conn)
		}
	}
}

func (s *Server) serveConn(conn *net.TCPConn) {
	defer conn.Close()

	// handshake
	var header [4]byte
	_, err := conn.Read(header[:])
	if err != nil {
		log.Println("server: handshake failed", err)
		return
	}

	if header != ([4]byte{'L', 'G', 'O', PROTO_VERSION}) {
		log.Println("server: invalid header")
		return
	}

	render := CreateRemoteRender(conn)
	pid := s.dispatcher.AttachPlayer(render)
	log.Printf("server: connection from '%s' now bound to user %d", conn.RemoteAddr(), pid)
	err = render.Run()
	if err != nil {
		log.Println("server: remote render error:", err)
	}

	s.dispatcher.DetachPlayer(pid)
	log.Printf("server: connection from '%s' have ended", conn.RemoteAddr())
}
