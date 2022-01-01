package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"testing"
)

func init() {
	// Ensure the goroutine server started
	_, err := startServer(nil)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
}

func TestServerGet(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:11211")
	if err != nil {
		t.Fatalf("cannot dial host: %v", err)
	}

	fmt.Fprintf(conn, "GET\r\n")

	resp, err := ioutil.ReadAll(conn)
	if err != nil {
		t.Fatalf("cannot read: %v", err)
	}

	if string(resp) != "OK" {
		t.Fatalf("unexpected response: %v", string(resp))
	}

	conn.Close()
}

func TestServerUnknownCommand(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:11211")
	if err != nil {
		t.Fatalf("cannot dial host: %v", err)
	}

	fmt.Fprintf(conn, "FOO\r\n")

	resp, err := ioutil.ReadAll(conn)
	if err != nil {
		t.Fatalf("cannot read: %v", err)
	}

	if string(resp) != "Unknown command" {
		t.Fatalf("unexpected response: %v", string(resp))
	}

	conn.Close()
}
