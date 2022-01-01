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
	data       map[string][]byte
}

func startServer(data map[string][]byte) (server, error) {
	listener, err := net.Listen("tcp", "localhost:11211")
	if err != nil {
		return server{}, err
	}

	srv := server{
		listener:   listener,
		requests:   sync.WaitGroup{},
		done:       make(chan interface{}),
		terminated: make(chan interface{}),
		data:       data,
	}

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
	return srv, nil
}

func (srv server) shutdown() {
	close(srv.done)
	<-srv.terminated
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

// TODO: decouple domain logic and network adapter
// This method should just connect them, but not pass conn to domain logic
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
		srv.doGet(conn, req)
	}
}

func (srv server) doGet(conn net.Conn, req request) {
	val, ok := srv.data[req.key]
	if ok {
		conn.Write(val)
		respond(conn, "") // write \r\n
	}
	fmt.Fprintf(conn, "ENDS\r\n")
}

type command int

const (
	commandGet command = iota
)

type request struct {
	cm  command
	key string
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

	parts := bytes.SplitN(buf, []byte(" "), 2)
	cm := string(parts[0])

	switch cm {
	case "GET":
		return parseRequestGet(parts[1:])
	default:
		return request{}, errors.New("unknown command")
	}
}

// TODO: support multiple keys
func parseRequestGet(parts [][]byte) (request, error) {
	if len(parts) == 0 {
		return request{}, errors.New("no key")
	}

	return request{
		cm:  commandGet,
		key: string(parts[0]),
	}, nil
}

func waitSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-sig
}

func main() {
	data := make(map[string][]byte)
	server, err := startServer(data)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}

	waitSignal()
	server.shutdown()
}
