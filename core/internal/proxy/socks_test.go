package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type dialerFunc func(context.Context, Target) (io.ReadWriteCloser, error)

func (f dialerFunc) DialContext(ctx context.Context, target Target) (io.ReadWriteCloser, error) {
	return f(ctx, target)
}

func TestReadSOCKSRequestStillSupportsSOCKS5(t *testing.T) {
	var raw bytes.Buffer
	raw.Write([]byte{5, 1, 0})
	raw.Write([]byte{5, 1, 0, socksAddrDomain, byte(len("example.com"))})
	raw.WriteString("example.com")
	raw.Write([]byte{0x01, 0xbb})
	var reply bytes.Buffer

	target, err := readSOCKSRequest(&raw, &reply)
	if err != nil {
		t.Fatal(err)
	}
	if target.Host != "example.com" || target.Port != 443 {
		t.Fatalf("target mismatch: got %+v", target)
	}
	if got, want := reply.Bytes(), []byte{5, 0}; !bytes.Equal(got, want) {
		t.Fatalf("SOCKS greeting reply mismatch: got %v want %v", got, want)
	}
}

func TestHTTPConnectUsesMixedInbound(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	errCh := make(chan error, 1)
	proxyServer := &SOCKSServer{
		Dialer: dialerFunc(func(ctx context.Context, target Target) (io.ReadWriteCloser, error) {
			if target.Host != "example.com" || target.Port != 443 {
				return nil, fmt.Errorf("target mismatch: got %+v", target)
			}
			upstreamClient, upstreamServer := net.Pipe()
			go func() {
				defer upstreamServer.Close()
				buf := make([]byte, 4)
				if _, err := io.ReadFull(upstreamServer, buf); err != nil {
					errCh <- err
					return
				}
				if string(buf) != "ping" {
					errCh <- fmt.Errorf("payload mismatch: %q", string(buf))
					return
				}
				_, err := upstreamServer.Write([]byte("pong"))
				errCh <- err
			}()
			return upstreamClient, nil
		}),
	}
	go proxyServer.handleConn(context.Background(), server)

	if _, err := io.WriteString(client, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(client)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "200 Connection Established") {
		t.Fatalf("CONNECT status mismatch: %q", status)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
	}
	if _, err := io.WriteString(client, "ping"); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "pong" {
		t.Fatalf("tunnel reply mismatch: %q", string(got))
	}
	assertNoAsyncError(t, errCh)
}

func TestHTTPAbsoluteFormRequestUsesMixedInbound(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	errCh := make(chan error, 1)
	proxyServer := &SOCKSServer{
		Dialer: dialerFunc(func(ctx context.Context, target Target) (io.ReadWriteCloser, error) {
			if target.Host != "example.com" || target.Port != 80 {
				return nil, fmt.Errorf("target mismatch: got %+v", target)
			}
			upstreamClient, upstreamServer := net.Pipe()
			go func() {
				defer upstreamServer.Close()
				req, err := http.ReadRequest(bufio.NewReader(upstreamServer))
				if err != nil {
					errCh <- err
					return
				}
				if req.RequestURI != "/path?q=1" {
					errCh <- fmt.Errorf("request uri mismatch: %q", req.RequestURI)
					return
				}
				if req.Host != "example.com" {
					errCh <- fmt.Errorf("host mismatch: %q", req.Host)
					return
				}
				_, err = io.WriteString(upstreamServer, "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
				errCh <- err
			}()
			return upstreamClient, nil
		}),
	}
	go proxyServer.handleConn(context.Background(), server)

	if _, err := io.WriteString(client, "GET http://example.com/path?q=1 HTTP/1.1\r\nHost: example.com\r\nProxy-Connection: keep-alive\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d", resp.StatusCode)
	}
	assertNoAsyncError(t, errCh)
}

func assertNoAsyncError(t *testing.T, errCh <-chan error) {
	t.Helper()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for proxy test goroutine")
	}
}
