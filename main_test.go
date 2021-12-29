package main

import (
	"io/ioutil"
	"net"
	"testing"
)

func init() {
	// Ensure the goroutine server started
	wg := startServer()
	wg.Wait()
}

func TestServer(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:11211")
	if err != nil {
		t.Fatalf("cannot dial host: %v", err)
	}

	resp, err := ioutil.ReadAll(conn)
	if err != nil {
		t.Fatalf("cannot read: %v", err)
	}

	if string(resp) != "OK" {
		t.Fatalf("unexpected response: %v", string(resp))
	}

	conn.Close()
}
