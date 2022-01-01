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

func startServer(done <-chan interface{}) (<-chan interface{}, error) {
	listener, err := net.Listen("tcp", "localhost:11211")
	if err != nil {
		return nil, err
	}

	var starter sync.WaitGroup
	starter.Add(2)

	var requests sync.WaitGroup
	terminated := make(chan interface{})

	// goroutine to control gracefull shutdown
	go func() {
		starter.Done()

		<-done
		listener.Close()
		requests.Wait()
		close(terminated)
	}()

	// goroutine to handle requests
	go func() {
		starter.Done()

		handleRequests(listener, &requests)
	}()

	starter.Wait()
	return terminated, nil
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

func handleRequest(conn net.Conn) {
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

func handleRequests(listener net.Listener, requests *sync.WaitGroup) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("cannot accept connection: %v", err)
			continue
		}

		requests.Add(1)
		handleRequest(conn)
		conn.Close()
		requests.Done()
	}
}

func waitSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-sig
}

func main() {
	done := make(chan interface{})
	terminated, err := startServer(done)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}

	waitSignal()
	close(done)
	<-terminated
}
