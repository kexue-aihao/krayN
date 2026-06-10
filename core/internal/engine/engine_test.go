package engine

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"io"
	"net"
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
