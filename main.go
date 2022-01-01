package main

import (
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

func handleRequests(listener net.Listener, requests *sync.WaitGroup) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("cannot accept connection: %v", err)
			continue
		}

		requests.Add(1)

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			requests.Done()
			continue
		}

		switch string(buf[:n]) {
		case "GET\r\n":
			fmt.Fprintf(conn, "OK")
		default:
			fmt.Fprintf(conn, "CLIENT_ERROR unknown command")
		}

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
