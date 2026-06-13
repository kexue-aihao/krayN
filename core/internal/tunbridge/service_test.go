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

func TestKrayProxyDialUDPDropsLocalDiscoveryTraffic(t *testing.T) {
	dialed := false
	proxy := &krayProxy{dialer: kraytun.DialFunc(func(context.Context) (io.ReadWriteCloser, error) {
		dialed = true
		return nil, nil
	})}

	tests := []struct {
		name string
		ip   string
		port uint16
	}{
		{name: "tun broadcast netbios ns", ip: "172.18.0.3", port: 137},
		{name: "tun broadcast netbios datagram", ip: "172.18.0.3", port: 138},
		{name: "limited broadcast", ip: "255.255.255.255", port: 9},
		{name: "mdns multicast", ip: "224.0.0.251", port: 5353},
		{name: "llmnr multicast", ip: "224.0.0.252", port: 5355},
		{name: "ssdp multicast", ip: "239.255.255.250", port: 1900},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialed = false
			packetConn, err := proxy.DialUDP(&M.Metadata{
				DstIP:   mustAddr(t, tt.ip),
				DstPort: tt.port,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer packetConn.Close()
			if dialed {
				t.Fatal("local discovery traffic should not dial kray udp associate")
			}
			n, err := packetConn.WriteTo([]byte("drop"), &net.UDPAddr{IP: net.ParseIP(tt.ip), Port: int(tt.port)})
			if err != nil {
				t.Fatal(err)
			}
			if n != len("drop") {
				t.Fatalf("blackhole write mismatch: got %d", n)
			}
		})
	}
}

func TestKrayProxyDialUDPAllowsNormalTraffic(t *testing.T) {
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
		errCh <- relay.WriteResponse(server, relay.Response{Status: relay.StatusOK})
	}()

	proxy := &krayProxy{dialer: kraytun.DialFunc(func(context.Context) (io.ReadWriteCloser, error) {
		return client, nil
	})}
	packetConn, err := proxy.DialUDP(&M.Metadata{
		DstIP:   mustAddr(t, "8.8.8.8"),
		DstPort: 53,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer packetConn.Close()
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
