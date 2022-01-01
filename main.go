package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type server struct {
	listener   net.Listener
	requests   sync.WaitGroup
	done       chan interface{}
	terminated chan interface{}
}

func newServer() (server, error) {
	listener, err := net.Listen("tcp", "localhost:11211")
	if err != nil {
		return server{}, err
	}

	return server{
		listener:   listener,
		requests:   sync.WaitGroup{},
		done:       make(chan interface{}),
		terminated: make(chan interface{}),
	}, nil
}

func (srv server) shutdown() {
	close(srv.done)
	<-srv.terminated
}

func (srv server) start() {
	var starter sync.WaitGroup
	starter.Add(2)

	// goroutine to control gracefull shutdown
	go func() {
		starter.Done()

		<-srv.done
		srv.listener.Close()
		srv.requests.Wait()
		close(srv.terminated)
	}()

	// goroutine to handle requests
	go func() {
		starter.Done()

		srv.handleRequests()
	}()

	starter.Wait()
}

func (srv server) handleRequests() {
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			log.Printf("cannot accept connection: %v", err)
			continue
		}

		srv.requests.Add(1)
		srv.handleRequest(conn)
		srv.requests.Done()
	}
}

func (srv server) handleRequest(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	req, err := parseRequest(buf[:n])
	if err != nil {
		respond(conn, "CLIENT_ERROR %v", err)
		return
	}

	switch req.cm {
	case commandGet:
		respond(conn, "OK")
	}
}

type command int

const (
	commandGet command = iota
)

type request struct {
	cm command
}

func respond(conn net.Conn, format string, a ...interface{}) {
	format = fmt.Sprintf("%s\r\n", format)
	fmt.Fprintf(conn, format, a...)
}

func parseRequest(buf []byte) (request, error) {
	if !bytes.HasSuffix(buf, []byte("\r\n")) {
		return request{}, errors.New("malformed request")
	}
	buf = buf[0 : len(buf)-2]

	switch {
	case bytes.HasPrefix(buf, []byte("GET")):
		return request{commandGet}, nil
	default:
		return request{}, errors.New("unknown command")
	}
}

func waitSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-sig
}

func main() {
	server, err := newServer()
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}

	server.start()
	waitSignal()
	server.shutdown()
}
