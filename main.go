package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

func startServer() *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		server, err := net.Listen("tcp", "localhost:11211")
		if err != nil {
			log.Fatalf("cannot listen: %v", err)
		}
		defer server.Close()

		wg.Done()

		for {
			conn, err := server.Accept()
			if err != nil {
				log.Printf("cannot accept connection: %v", err)
				continue
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				conn.Close()
				continue
			}

			switch string(buf[:n]) {
			case "GET\r\n":
				fmt.Fprintf(conn, "OK")
			default:
				fmt.Fprintf(conn, "Unknown command")
			}

			conn.Close()
		}
	}()

	return &wg
}
