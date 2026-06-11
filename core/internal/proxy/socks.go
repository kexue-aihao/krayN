package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"krayn/core/internal/stats"
)

type Dialer interface {
	DialContext(ctx context.Context, target Target) (io.ReadWriteCloser, error)
}

type SOCKSServer struct {
	Address string
	Dialer  Dialer
	Stats   *stats.Collector
	Logger  *slog.Logger

	listener net.Listener
	closeMu  sync.Mutex
}

func (s *SOCKSServer) Start(ctx context.Context) error {
	if s.Dialer == nil {
		return errors.New("proxy dialer is required")
	}
	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	s.closeMu.Lock()
	s.listener = listener
	s.closeMu.Unlock()
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *SOCKSServer) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}

func (s *SOCKSServer) handleConn(ctx context.Context, client net.Conn) {
	defer client.Close()
	if s.Stats != nil {
		s.Stats.OpenConnection()
		defer s.Stats.CloseConnection()
	}
	if err := client.SetDeadline(time.Now().Add(15 * time.Second)); err != nil && s.Logger != nil {
		s.Logger.Debug("set proxy handshake deadline", "error", err)
	}
	reader := bufio.NewReader(client)
	first, err := reader.Peek(1)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Debug("proxy request failed", "error", err)
		}
		return
	}

	if first[0] == 5 {
		s.handleSOCKSConn(ctx, client, reader)
		return
	}
	s.handleHTTPProxyConn(ctx, client, reader)
}

func (s *SOCKSServer) handleSOCKSConn(ctx context.Context, client net.Conn, reader *bufio.Reader) {
	target, err := readSOCKSRequest(reader, client)
	if err != nil {
		_ = writeSOCKSReply(client, 1)
		if s.Logger != nil {
			s.Logger.Debug("socks request failed", "error", err)
		}
		return
	}
	if err := client.SetDeadline(time.Time{}); err != nil && s.Logger != nil {
		s.Logger.Debug("clear socks deadline", "error", err)
	}
	upstream, err := s.Dialer.DialContext(ctx, target)
	if err != nil {
		_ = writeSOCKSReply(client, 5)
		if s.Logger != nil {
			s.Logger.Warn("outbound dial failed", "target", target.Address(), "error", err)
		}
		return
	}
	defer upstream.Close()
	if err := writeSOCKSReply(client, 0); err != nil {
		return
	}
	pipeBoth(client, reader, upstream, s.Stats)
}

func (s *SOCKSServer) handleHTTPProxyConn(ctx context.Context, client net.Conn, reader *bufio.Reader) {
	req, err := readHTTPProxyRequest(reader)
	if err != nil {
		_ = writeHTTPProxyError(client, http.StatusBadRequest, "Bad Request")
		if s.Logger != nil {
			s.Logger.Debug("http proxy request failed", "error", err)
		}
		return
	}
	if err := client.SetDeadline(time.Time{}); err != nil && s.Logger != nil {
		s.Logger.Debug("clear http proxy deadline", "error", err)
	}
	upstream, err := s.Dialer.DialContext(ctx, req.Target)
	if err != nil {
		_ = writeHTTPProxyError(client, http.StatusBadGateway, "Bad Gateway")
		if s.Logger != nil {
			s.Logger.Warn("outbound dial failed", "target", req.Target.Address(), "error", err)
		}
		return
	}
	defer upstream.Close()
	if req.IsConnect {
		if err := writeHTTPConnectOK(client); err != nil {
			return
		}
	} else if err := req.Write(upstream); err != nil {
		_ = writeHTTPProxyError(client, http.StatusBadGateway, "Bad Gateway")
		return
	}
	pipeBoth(client, reader, upstream, s.Stats)
}

func readSOCKSRequest(r io.Reader, w io.Writer) (Target, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Target{}, err
	}
	if header[0] != 5 {
		return Target{}, errors.New("only SOCKS5 is supported")
	}
	methodCount := int(header[1])
	if methodCount == 0 {
		return Target{}, errors.New("empty SOCKS method list")
	}
	methods := make([]byte, methodCount)
	if _, err := io.ReadFull(r, methods); err != nil {
		return Target{}, err
	}
	hasNoAuth := false
	for _, method := range methods {
		if method == 0 {
			hasNoAuth = true
			break
		}
	}
	if !hasNoAuth {
		if _, err := w.Write([]byte{5, 0xff}); err != nil {
			return Target{}, err
		}
		return Target{}, errors.New("SOCKS auth method unsupported")
	}
	if _, err := w.Write([]byte{5, 0}); err != nil {
		return Target{}, err
	}

	var req [4]byte
	if _, err := io.ReadFull(r, req[:]); err != nil {
		return Target{}, err
	}
	if req[0] != 5 || req[1] != 1 || req[2] != 0 {
		return Target{}, errors.New("only CONNECT command is supported")
	}
	var host string
	switch req[3] {
	case addrIPv4:
		var raw [4]byte
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return Target{}, err
		}
		host = net.IP(raw[:]).String()
	case addrIPv6:
		var raw [16]byte
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return Target{}, err
		}
		host = net.IP(raw[:]).String()
	case addrDomain:
		var size [1]byte
		if _, err := io.ReadFull(r, size[:]); err != nil {
			return Target{}, err
		}
		if size[0] == 0 {
			return Target{}, errors.New("empty SOCKS domain")
		}
		raw := make([]byte, int(size[0]))
		if _, err := io.ReadFull(r, raw); err != nil {
			return Target{}, err
		}
		host = string(raw)
	default:
		return Target{}, fmt.Errorf("unsupported SOCKS address type %d", req[3])
	}
	var port [2]byte
	if _, err := io.ReadFull(r, port[:]); err != nil {
		return Target{}, err
	}
	return Target{Host: host, Port: binary.BigEndian.Uint16(port[:])}, nil
}

func writeSOCKSReply(w io.Writer, status byte) error {
	_, err := w.Write([]byte{5, status, 0, addrIPv4, 0, 0, 0, 0, 0, 0})
	return err
}

type httpProxyRequest struct {
	Target    Target
	IsConnect bool
	Request   *http.Request
}

func (r httpProxyRequest) Write(w io.Writer) error {
	if r.Request == nil {
		return nil
	}
	return r.Request.Write(w)
}

func readHTTPProxyRequest(reader *bufio.Reader) (httpProxyRequest, error) {
	req, err := http.ReadRequest(reader)
	if err != nil {
		return httpProxyRequest{}, err
	}
	if strings.EqualFold(req.Method, http.MethodConnect) {
		authority := req.Host
		if authority == "" && req.URL != nil {
			authority = req.URL.Host
			if authority == "" {
				authority = req.URL.Opaque
			}
		}
		if authority == "" {
			authority = req.RequestURI
		}
		target, err := parseProxyTarget(authority, "443")
		if err != nil {
			return httpProxyRequest{}, err
		}
		return httpProxyRequest{Target: target, IsConnect: true}, nil
	}

	authority := req.Host
	defaultPort := "80"
	if req.URL != nil && req.URL.IsAbs() {
		authority = req.URL.Host
		switch strings.ToLower(req.URL.Scheme) {
		case "https":
			defaultPort = "443"
		default:
			defaultPort = "80"
		}
	}
	target, err := parseProxyTarget(authority, defaultPort)
	if err != nil {
		return httpProxyRequest{}, err
	}
	req.RequestURI = ""
	if req.URL != nil {
		req.URL.Scheme = ""
		req.URL.Host = ""
		req.URL.User = nil
	}
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Close = true
	req.Header.Set("Connection", "close")
	return httpProxyRequest{Target: target, Request: req}, nil
}

func parseProxyTarget(authority, defaultPort string) (Target, error) {
	authority = strings.TrimSpace(authority)
	if authority == "" {
		return Target{}, errors.New("proxy target is required")
	}
	if parsed, err := url.Parse(authority); err == nil && parsed.Host != "" {
		authority = parsed.Host
	}
	if host, port, err := net.SplitHostPort(authority); err == nil {
		return parseTargetHostPort(host, port)
	}
	if strings.HasPrefix(authority, "[") {
		end := strings.LastIndex(authority, "]")
		if end > 0 {
			host := authority[1:end]
			rest := strings.TrimSpace(authority[end+1:])
			if strings.HasPrefix(rest, ":") && len(rest) > 1 {
				return parseTargetHostPort(host, rest[1:])
			}
			if rest == "" && defaultPort != "" {
				return parseTargetHostPort(host, defaultPort)
			}
		}
	}
	if host, port, ok := strings.Cut(authority, ":"); ok && !strings.Contains(host, ":") && port != "" {
		return parseTargetHostPort(host, port)
	}
	if defaultPort == "" {
		return Target{}, fmt.Errorf("missing port in proxy target %q", authority)
	}
	return parseTargetHostPort(authority, defaultPort)
}

func parseTargetHostPort(host, port string) (Target, error) {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		return Target{}, errors.New("proxy target host is required")
	}
	return ParseTarget(net.JoinHostPort(host, port))
}

func writeHTTPConnectOK(w io.Writer) error {
	_, err := io.WriteString(w, "HTTP/1.1 200 Connection Established\r\n\r\n")
	return err
}

func writeHTTPProxyError(w io.Writer, status int, message string) error {
	body := message + "\n"
	_, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\nConnection: close\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\n\r\n%s", status, http.StatusText(status), len(body), body)
	return err
}

func pipeBoth(client net.Conn, clientReader io.Reader, upstream io.ReadWriteCloser, collector *stats.Collector) {
	done := make(chan struct{}, 2)
	go func() {
		copyAndCount(upstream, clientReader, func(n int64) {
			if collector != nil {
				collector.AddUploaded(n)
			}
		})
		_ = upstream.Close()
		_ = client.Close()
		done <- struct{}{}
	}()
	go func() {
		copyAndCount(client, upstream, func(n int64) {
			if collector != nil {
				collector.AddDownloaded(n)
			}
		})
		_ = upstream.Close()
		_ = client.Close()
		done <- struct{}{}
	}()
	<-done
}

func copyAndCount(dst io.Writer, src io.Reader, add func(int64)) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			add(int64(written))
			if writeErr != nil {
				return
			}
			if written != n {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
