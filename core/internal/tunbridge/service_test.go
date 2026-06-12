package tunbridge

import (
	"context"
	"io"
	"net"
	"net/netip"
	"testing"

	M "github.com/xjasonlyu/tun2socks/v2/metadata"
	"kray/pkg/relay"
	kraytun "kray/pkg/tun"
)

func TestKrayProxyDialTCPWritesRelayRequest(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		req, err := relay.ReadRequest(server)
		if err != nil {
			errCh <- err
			return
		}
		if req.Command != relay.CommandTCPConnect || req.Address.Host != "1.2.3.4" || req.Address.Port != 443 {
			t.Errorf("unexpected request: %+v", req)
		}
		errCh <- relay.WriteResponse(server, relay.Response{Status: relay.StatusOK})
	}()

	proxy := &krayProxy{dialer: kraytun.DialFunc(func(context.Context) (io.ReadWriteCloser, error) {
		return client, nil
	})}
	conn, err := proxy.DialContext(context.Background(), &M.Metadata{
		DstIP:   mustAddr(t, "1.2.3.4"),
		DstPort: 443,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestKrayProxyDialUDPWritesDatagram(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		req, err := relay.ReadRequest(server)
		if err != nil {
			errCh <- err
			return
		}
		if req.Command != relay.CommandUDPAssociate {
			t.Errorf("unexpected udp request: %+v", req)
		}
		if err := relay.WriteResponse(server, relay.Response{Status: relay.StatusOK}); err != nil {
			errCh <- err
			return
		}
		datagram, err := relay.ReadDatagram(server)
		if err != nil {
			errCh <- err
			return
		}
		if datagram.Address.Host != "8.8.8.8" || datagram.Address.Port != 53 || string(datagram.Payload) != "dns" {
			t.Errorf("unexpected datagram: %+v %q", datagram.Address, datagram.Payload)
		}
		errCh <- nil
	}()

	proxy := &krayProxy{dialer: kraytun.DialFunc(func(context.Context) (io.ReadWriteCloser, error) {
		return client, nil
	})}
	packetConn, err := proxy.DialUDP(&M.Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	defer packetConn.Close()
	if _, err := packetConn.WriteTo([]byte("dns"), &net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func mustAddr(t *testing.T, text string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(text)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}
