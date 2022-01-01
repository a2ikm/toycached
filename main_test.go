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
	server, err := newServer()
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
	server.start()
}

func TestServer(t *testing.T) {
	tests := []struct {
		name    string
		inReq   string
		outResp string
	}{
		{
			"Successful GET",
			"GET\r\n",
			"OK\r\n",
		},
		{
			"Non-CRLF GET",
			"GET",
			"CLIENT_ERROR malformed request\r\n",
		},
		{
			"Unknown command",
			"FOO\r\n",
			"CLIENT_ERROR unknown command\r\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, err := net.Dial("tcp", "localhost:11211")
			if err != nil {
				t.Fatalf("cannot dial host: %v", err)
			}

			fmt.Fprintf(conn, test.inReq)

			resp, err := ioutil.ReadAll(conn)
			if err != nil {
				t.Fatalf("cannot read: %v", err)
			}

			if string(resp) != test.outResp {
				t.Fatalf("unexpected response: %v", string(resp))
			}

			conn.Close()
		})
	}
}
