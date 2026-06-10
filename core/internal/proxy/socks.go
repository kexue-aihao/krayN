package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
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
		s.Logger.Debug("set socks handshake deadline", "error", err)
	}
	target, err := readSOCKSRequest(client)
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
	pipeBoth(client, upstream, s.Stats)
}

func readSOCKSRequest(rw io.ReadWriter) (Target, error) {
	var header [2]byte
	if _, err := io.ReadFull(rw, header[:]); err != nil {
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
	if _, err := io.ReadFull(rw, methods); err != nil {
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
		if _, err := rw.Write([]byte{5, 0xff}); err != nil {
			return Target{}, err
		}
		return Target{}, errors.New("SOCKS auth method unsupported")
	}
	if _, err := rw.Write([]byte{5, 0}); err != nil {
		return Target{}, err
	}

	var req [4]byte
	if _, err := io.ReadFull(rw, req[:]); err != nil {
		return Target{}, err
	}
	if req[0] != 5 || req[1] != 1 || req[2] != 0 {
		return Target{}, errors.New("only CONNECT command is supported")
	}
	var host string
	switch req[3] {
	case addrIPv4:
		var raw [4]byte
		if _, err := io.ReadFull(rw, raw[:]); err != nil {
			return Target{}, err
		}
		host = net.IP(raw[:]).String()
	case addrIPv6:
		var raw [16]byte
		if _, err := io.ReadFull(rw, raw[:]); err != nil {
			return Target{}, err
		}
		host = net.IP(raw[:]).String()
	case addrDomain:
		var size [1]byte
		if _, err := io.ReadFull(rw, size[:]); err != nil {
			return Target{}, err
		}
		if size[0] == 0 {
			return Target{}, errors.New("empty SOCKS domain")
		}
		raw := make([]byte, int(size[0]))
		if _, err := io.ReadFull(rw, raw); err != nil {
			return Target{}, err
		}
		host = string(raw)
	default:
		return Target{}, fmt.Errorf("unsupported SOCKS address type %d", req[3])
	}
	var port [2]byte
	if _, err := io.ReadFull(rw, port[:]); err != nil {
		return Target{}, err
	}
	return Target{Host: host, Port: binary.BigEndian.Uint16(port[:])}, nil
}

func writeSOCKSReply(w io.Writer, status byte) error {
	_, err := w.Write([]byte{5, status, 0, addrIPv4, 0, 0, 0, 0, 0, 0})
	return err
}

func pipeBoth(client net.Conn, upstream io.ReadWriteCloser, collector *stats.Collector) {
	done := make(chan struct{}, 2)
	go func() {
		copyAndCount(upstream, client, func(n int64) {
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
