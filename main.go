package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

	in := make([]byte, 1024)
	n, err := conn.Read(in)
	if err != nil {
		return
	}

	out := process(in[:n], srv.data)
	conn.Write(out)
}

func process(in []byte, data map[string][]byte) []byte {
	var buf bytes.Buffer

	req, err := parseRequest(in)
	if err != nil {
		fmt.Fprintf(&buf, "CLIENT_ERROR %v\r\n", err)
		return buf.Bytes()
	}

	switch req.cm {
	case commandGet:
		doGet(&buf, req, data)
	}

	return buf.Bytes()
}

func doGet(out io.Writer, req request, data map[string][]byte) {
	val, ok := data[req.key]
	if ok {
		out.Write(val)
		fmt.Fprintf(out, "\r\n")
	}
	fmt.Fprintf(out, "ENDS\r\n")
}

type command int

const (
	commandGet command = iota
)

type request struct {
	cm  command
	key string
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
