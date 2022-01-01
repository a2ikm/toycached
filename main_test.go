package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"testing"
)

func init() {
	data := map[string][]byte{
		"foo": []byte("foo value"),
	}
	_, err := startServer(data)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
}

func TestServer(t *testing.T) {
	tests := []struct {
		name    string
		inReq   string
		outResp string
	}{
		{
			"Successful GET",
			"GET foo\r\n",
			"foo value\r\nENDS\r\n",
		},
		{
			"Non-existing GET",
			"GET bar\r\n",
			"ENDS\r\n",
		},
		{
			"CRLF-less GET",
			"GET foo",
			"CLIENT_ERROR malformed request\r\n",
		},
		{
			"Key-less GET",
			"GET\r\n",
			"CLIENT_ERROR no key\r\n",
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
