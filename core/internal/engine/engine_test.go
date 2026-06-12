package engine

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"kray/pkg/kless"
	"krayn/core/internal/config"
	"krayn/core/internal/proxy"
)

func TestDialContextOverKLESS(t *testing.T) {
	serverPublic, serverPrivate, err := kless.GenerateServerIdentity()
	if err != nil {
		t.Fatal(err)
	}
	clientSecret, err := kless.GenerateClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer targetListener.Close()
	go func() {
		conn, err := targetListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	klessListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer klessListener.Close()
	go serveRelay(t, klessListener, serverPrivate, clientSecret)

	cfg := config.Default()
	profile := config.UpsertProfile(&cfg, config.Profile{
		Name:            "test",
		Transport:       "tcp",
		Endpoint:        klessListener.Addr().String(),
		ClientID:        "test-client",
		ClientSecret:    kless.EncodeKey(clientSecret),
		ServerPublicKey: kless.EncodeKey(serverPublic),
	})
	cfg.ActiveProfileID = profile.ID
	appEngine := &Engine{
		cfg:    cfg,
		stats:  nil,
		status: StatusStopped,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	target, err := proxy.ParseTarget(targetListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn, err := appEngine.DialContext(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("ping")) {
		t.Fatalf("got %q", string(got))
	}
}

func TestDialContextAcceptsURLSafeKLESSKeys(t *testing.T) {
	serverPublic, serverPrivate, err := kless.GenerateServerIdentity()
	if err != nil {
		t.Fatal(err)
	}
	clientSecret, err := kless.GenerateClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer targetListener.Close()
	go func() {
		conn, err := targetListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	klessListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer klessListener.Close()
	go serveRelay(t, klessListener, serverPrivate, clientSecret)

	cfg := config.Default()
	profile := config.UpsertProfile(&cfg, config.Profile{
		Name:            "test",
		Transport:       "tcp",
		Endpoint:        klessListener.Addr().String(),
		ClientID:        "test-client",
		ClientSecret:    base64.RawURLEncoding.EncodeToString(clientSecret),
		ServerPublicKey: base64.URLEncoding.EncodeToString(serverPublic),
	})
	cfg.ActiveProfileID = profile.ID
	appEngine := &Engine{
		cfg:    cfg,
		stats:  nil,
		status: StatusStopped,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	target, err := proxy.ParseTarget(targetListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn, err := appEngine.DialContext(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("pong")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("pong")) {
		t.Fatalf("got %q", string(got))
	}
}

func TestDialContextReportsKnodeRelayHintOnHandshakeEOF(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}()

	serverPublic, _, err := kless.GenerateServerIdentity()
	if err != nil {
		t.Fatal(err)
	}
	clientSecret, err := kless.GenerateClientSecret()
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	profile := config.UpsertProfile(&cfg, config.Profile{
		Name:            "eof",
		Transport:       "tcp",
		Endpoint:        listener.Addr().String(),
		ClientID:        "client-1",
		ClientSecret:    kless.EncodeKey(clientSecret),
		ServerPublicKey: kless.EncodeKey(serverPublic),
	})
	cfg.ActiveProfileID = profile.ID
	appEngine := &Engine{
		cfg:    cfg,
		stats:  nil,
		status: StatusStopped,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	target, err := proxy.ParseTarget("www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	_, err = appEngine.DialContext(ctx, target)
	if err == nil {
		t.Fatal("expected dial failure")
	}
	if !strings.Contains(err.Error(), "kless-server") || !strings.Contains(err.Error(), "local-tcp") {
		t.Fatalf("missing Knode hint: %v", err)
	}
}

func TestDecodeKLESSKeyAcceptsWhitespaceAndURLSafeEncoding(t *testing.T) {
	original := []byte("0123456789abcdef0123456789abcdef")
	cases := []struct {
		name string
		text string
	}{
		{
			name: "raw_std",
			text: "  " + base64.RawStdEncoding.EncodeToString(original) + "\n",
		},
		{
			name: "raw_url",
			text: "\t" + base64.RawURLEncoding.EncodeToString(original) + "\r\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeKLESSKey(tc.text)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, original) {
				t.Fatalf("got %q", string(got))
			}
		})
	}
}

func TestResolveProxyTargetPrefersIPv4(t *testing.T) {
	target := proxy.Target{Host: "example.com", Port: 443}
	resolved, err := resolveProxyTarget(context.Background(), target, stubIPResolver{
		ips: []net.IP{
			net.ParseIP("2001:db8::1"),
			net.ParseIP("1.2.3.4"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Host != "1.2.3.4" {
		t.Fatalf("got %q", resolved.Host)
	}
	if resolved.Port != target.Port {
		t.Fatalf("port changed: %d", resolved.Port)
	}
}

func TestResolveProxyTargetKeepsIPTargets(t *testing.T) {
	target := proxy.Target{Host: "1.2.3.4", Port: 443}
	resolved, err := resolveProxyTarget(context.Background(), target, stubIPResolver{
		ips: []net.IP{net.ParseIP("9.9.9.9")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Host != target.Host || resolved.Port != target.Port {
		t.Fatalf("target changed: %#v", resolved)
	}
}

type stubIPResolver struct {
	ips []net.IP
	err error
}

func (s stubIPResolver) LookupIP(context.Context, string) ([]net.IP, error) {
	return append([]net.IP(nil), s.ips...), s.err
}

func serveRelay(t *testing.T, listener net.Listener, private ed25519.PrivateKey, clientSecret []byte) {
	t.Helper()
	raw, err := listener.Accept()
	if err != nil {
		return
	}
	defer raw.Close()
	conn, _, err := kless.ServerHandshake(raw, kless.ServerConfig{
		SigningKey:  private,
		ClientStore: kless.StaticClientStore{"test-client": clientSecret},
	})
	if err != nil {
		t.Errorf("server handshake: %v", err)
		return
	}
	defer conn.Close()
	target, err := proxy.ReadConnectRequest(conn)
	if err != nil {
		t.Errorf("read connect request: %v", err)
		return
	}
	upstream, err := net.Dial("tcp", target.Address())
	if err != nil {
		_ = proxy.WriteConnectResponse(conn, err.Error())
		t.Errorf("dial target: %v", err)
		return
	}
	defer upstream.Close()
	if err := proxy.WriteConnectResponse(conn, ""); err != nil {
		t.Errorf("write connect response: %v", err)
		return
	}
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstream, conn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(conn, upstream)
		done <- struct{}{}
	}()
	<-done
}
