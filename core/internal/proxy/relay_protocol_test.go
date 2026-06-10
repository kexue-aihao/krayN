package proxy

import (
	"bytes"
	"testing"
)

func TestConnectRequestRoundTrip(t *testing.T) {
	targets := []Target{
		{Host: "example.com", Port: 443},
		{Host: "127.0.0.1", Port: 8080},
		{Host: "::1", Port: 8081},
	}
	for _, target := range targets {
		var buf bytes.Buffer
		if err := WriteConnectRequest(&buf, target); err != nil {
			t.Fatal(err)
		}
		got, err := ReadConnectRequest(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if got.Host != target.Host || got.Port != target.Port {
			t.Fatalf("got %+v want %+v", got, target)
		}
	}
}
