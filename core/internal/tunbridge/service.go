package tunbridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"kray/pkg/relay"
	kraytun "kray/pkg/tun"
	"krayn/core/internal/config"

	tun2core "github.com/xjasonlyu/tun2socks/v2/core"
	tundevice "github.com/xjasonlyu/tun2socks/v2/core/device/tun"
	M "github.com/xjasonlyu/tun2socks/v2/metadata"
	tun2proxy "github.com/xjasonlyu/tun2socks/v2/proxy"
	"github.com/xjasonlyu/tun2socks/v2/proxy/proto"
	"github.com/xjasonlyu/tun2socks/v2/tunnel"
	"github.com/xjasonlyu/tun2socks/v2/tunnel/statistic"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Service struct {
	device interface {
		Name() string
		Close()
	}
	routes *routeManager
	stack  *stack.Stack
	tunnel *tunnel.Tunnel
	logger *slog.Logger

	closeOnce sync.Once
}

type Option struct {
	Config      config.TunConfig
	Dialer      kraytun.Dialer
	EndpointIPs []net.IP
	Logger      *slog.Logger
}

func Start(ctx context.Context, option Option) (*Service, error) {
	if !option.Config.Enabled {
		return nil, nil
	}
	if option.Dialer == nil {
		return nil, kraytun.ErrNilDialer
	}
	cfg := config.NormalizeTun(option.Config)
	if err := config.ValidateTun(cfg); err != nil {
		return nil, err
	}
	logger := option.Logger
	if logger == nil {
		logger = slog.Default()
	}

	proxy := &krayProxy{dialer: option.Dialer}
	tun := tunnel.New(proxy, statistic.DefaultManager)
	tun.SetUDPTimeout(time.Duration(cfg.UDPTimeoutSeconds) * time.Second)
	prepareTunDevice(cfg.InterfaceName)
	dev, err := tundevice.Open(cfg.InterfaceName, uint32(cfg.MTU))
	if err != nil {
		return nil, fmt.Errorf("open tun device: %w", err)
	}
	st, err := tun2core.CreateStack(&tun2core.Config{
		LinkEndpoint:     dev,
		TransportHandler: tun,
	})
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("create tun stack: %w", err)
	}

	service := &Service{
		device: dev,
		stack:  st,
		tunnel: tun,
		logger: logger,
	}
	if cfg.AutoRoute {
		routes, err := setupAutoRoute(routeOptions{
			InterfaceName: dev.Name(),
			MTU:           cfg.MTU,
			EndpointIPs:   option.EndpointIPs,
			ExcludeCIDRs:  cfg.RouteExclude,
			Logger:        logger,
		})
		if err != nil {
			service.Close()
			return nil, fmt.Errorf("setup tun auto route: %w", err)
		}
		service.routes = routes
	}
	tun.ProcessAsync()
	go func() {
		<-ctx.Done()
		service.Close()
	}()
	logger.Info("tun bridge started", "interface", dev.Name(), "mtu", cfg.MTU)
	return service, nil
}

func (s *Service) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.tunnel != nil {
			s.tunnel.Close()
		}
		if s.routes != nil {
			s.routes.Close()
		}
		if s.device != nil {
			s.device.Close()
		}
		if s.stack != nil {
			s.stack.Close()
			s.stack.Wait()
		}
		if s.logger != nil {
			s.logger.Info("tun bridge stopped")
		}
	})
}

func (s *Service) InterfaceName() string {
	if s == nil || s.device == nil {
		return ""
	}
	return s.device.Name()
}

type krayProxy struct {
	dialer kraytun.Dialer
}

var _ tun2proxy.Proxy = (*krayProxy)(nil)

func (p *krayProxy) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	target, err := relayAddressFromMetadata(metadata)
	if err != nil {
		return nil, err
	}
	conn, err := kraytun.DialTCP(ctx, p.dialer, target)
	if err != nil {
		return nil, err
	}
	return &connAdapter{
		ReadWriteCloser: conn,
		remoteAddr:      relayNetAddr(target),
	}, nil
}

func (p *krayProxy) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	if shouldDropLocalUDP(metadata) {
		return newBlackholePacketConn(metadata), nil
	}
	assoc, err := kraytun.AssociateUDP(context.Background(), p.dialer)
	if err != nil {
		return nil, err
	}
	local := relayNetAddr(relay.Address{Host: "0.0.0.0"})
	if metadata != nil && metadata.MidIP.IsValid() {
		local = &net.UDPAddr{IP: net.IP(metadata.MidIP.AsSlice()), Port: int(metadata.MidPort)}
	}
	return &packetConnAdapter{
		assoc: assoc,
		local: local,
	}, nil
}

func (p *krayProxy) Addr() string {
	return "kray"
}

func (p *krayProxy) Proto() proto.Proto {
	return proto.Relay
}

func shouldDropLocalUDP(metadata *M.Metadata) bool {
	if metadata == nil || !metadata.DstIP.IsValid() {
		return false
	}
	if isLocalDiscoveryUDP(metadata.DstPort) {
		return true
	}
	addr := metadata.DstIP
	if addr.IsMulticast() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsInterfaceLocalMulticast() {
		return true
	}
	if addr.Is4() {
		if addr == netip.MustParseAddr("255.255.255.255") || addr == netip.MustParseAddr("172.18.0.3") {
			return true
		}
	}
	return false
}

func isLocalDiscoveryUDP(port uint16) bool {
	switch port {
	case 137, // NetBIOS Name Service
		138,  // NetBIOS Datagram Service
		5353, // mDNS
		5355, // LLMNR
		1900: // SSDP
		return true
	default:
		return false
	}
}

func relayAddressFromMetadata(metadata *M.Metadata) (relay.Address, error) {
	if metadata == nil {
		return relay.Address{}, errors.New("tun metadata is nil")
	}
	if !metadata.DstIP.IsValid() {
		return relay.Address{}, errors.New("tun destination ip is invalid")
	}
	if metadata.DstPort == 0 {
		return relay.Address{}, errors.New("tun destination port is invalid")
	}
	return relay.Address{
		Host: metadata.DstIP.String(),
		Port: metadata.DstPort,
	}, nil
}

type connAdapter struct {
	io.ReadWriteCloser
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *connAdapter) LocalAddr() net.Addr {
	if c.localAddr != nil {
		return c.localAddr
	}
	return relayNetAddr(relay.Address{Host: "127.0.0.1"})
}

func (c *connAdapter) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return relayNetAddr(relay.Address{Host: "0.0.0.0"})
}

func (c *connAdapter) SetDeadline(t time.Time) error {
	if setter, ok := c.ReadWriteCloser.(interface{ SetDeadline(time.Time) error }); ok {
		return setter.SetDeadline(t)
	}
	return nil
}

func (c *connAdapter) SetReadDeadline(t time.Time) error {
	if setter, ok := c.ReadWriteCloser.(interface{ SetReadDeadline(time.Time) error }); ok {
		return setter.SetReadDeadline(t)
	}
	return nil
}

func (c *connAdapter) SetWriteDeadline(t time.Time) error {
	if setter, ok := c.ReadWriteCloser.(interface{ SetWriteDeadline(time.Time) error }); ok {
		return setter.SetWriteDeadline(t)
	}
	return nil
}

func (c *connAdapter) CloseRead() error {
	if closer, ok := c.ReadWriteCloser.(interface{ CloseRead() error }); ok {
		return closer.CloseRead()
	}
	return nil
}

func (c *connAdapter) CloseWrite() error {
	if closer, ok := c.ReadWriteCloser.(interface{ CloseWrite() error }); ok {
		return closer.CloseWrite()
	}
	return nil
}

type packetConnAdapter struct {
	assoc *kraytun.UDPAssociate
	local net.Addr

	readDeadlineMu sync.Mutex
	readDeadline   time.Time
	writeDeadline  time.Time
}

func (p *packetConnAdapter) ReadFrom(buf []byte) (int, net.Addr, error) {
	n, addr, err := p.assoc.ReadFrom(buf)
	return n, relayNetAddr(addr), err
}

func (p *packetConnAdapter) WriteTo(buf []byte, addr net.Addr) (int, error) {
	target, err := relayAddressFromNetAddr(addr)
	if err != nil {
		return 0, err
	}
	return p.assoc.WriteTo(buf, target)
}

func (p *packetConnAdapter) Close() error {
	if p.assoc == nil {
		return nil
	}
	return p.assoc.Close()
}

func (p *packetConnAdapter) LocalAddr() net.Addr {
	if p.local != nil {
		return p.local
	}
	return relayNetAddr(relay.Address{Host: "0.0.0.0"})
}

func (p *packetConnAdapter) SetDeadline(t time.Time) error {
	_ = p.SetReadDeadline(t)
	return p.SetWriteDeadline(t)
}

func (p *packetConnAdapter) SetReadDeadline(t time.Time) error {
	p.readDeadlineMu.Lock()
	p.readDeadline = t
	p.readDeadlineMu.Unlock()
	return nil
}

func (p *packetConnAdapter) SetWriteDeadline(t time.Time) error {
	p.readDeadlineMu.Lock()
	p.writeDeadline = t
	p.readDeadlineMu.Unlock()
	return nil
}

func relayAddressFromNetAddr(addr net.Addr) (relay.Address, error) {
	if addr == nil {
		return relay.Address{}, kraytun.ErrNilTarget
	}
	switch v := addr.(type) {
	case *net.UDPAddr:
		return relay.Address{Host: v.IP.String(), Port: uint16(v.Port)}, nil
	case *net.TCPAddr:
		return relay.Address{Host: v.IP.String(), Port: uint16(v.Port)}, nil
	case *M.Addr:
		return relayAddressFromMetadata(v.Metadata())
	default:
		host, portText, err := net.SplitHostPort(addr.String())
		if err != nil {
			return relay.Address{}, err
		}
		port, err := strconv.ParseUint(portText, 10, 16)
		if err != nil {
			return relay.Address{}, err
		}
		return relay.Address{Host: host, Port: uint16(port)}, nil
	}
}

func relayNetAddr(addr relay.Address) net.Addr {
	if ip, ok := parseNetIP(addr.Host); ok {
		return &net.UDPAddr{IP: ip, Port: int(addr.Port)}
	}
	if addr.Host == "" {
		addr.Host = "0.0.0.0"
	}
	return hostPortAddr{network: "kray", host: addr.Host, port: addr.Port}
}

func parseNetIP(host string) (net.IP, bool) {
	parsed, err := netip.ParseAddr(host)
	if err != nil {
		return nil, false
	}
	return net.IP(parsed.AsSlice()), true
}

type hostPortAddr struct {
	network string
	host    string
	port    uint16
}

func (a hostPortAddr) Network() string {
	if a.network != "" {
		return a.network
	}
	return "kray"
}

func (a hostPortAddr) String() string {
	return net.JoinHostPort(a.host, strconv.Itoa(int(a.port)))
}

type blackholePacketConn struct {
	local net.Addr
}

func newBlackholePacketConn(metadata *M.Metadata) *blackholePacketConn {
	local := relayNetAddr(relay.Address{Host: "0.0.0.0"})
	if metadata != nil && metadata.MidIP.IsValid() {
		local = &net.UDPAddr{IP: net.IP(metadata.MidIP.AsSlice()), Port: int(metadata.MidPort)}
	}
	return &blackholePacketConn{local: local}
}

func (p *blackholePacketConn) ReadFrom([]byte) (int, net.Addr, error) {
	return 0, nil, io.EOF
}

func (p *blackholePacketConn) WriteTo(buf []byte, _ net.Addr) (int, error) {
	return len(buf), nil
}

func (p *blackholePacketConn) Close() error {
	return nil
}

func (p *blackholePacketConn) LocalAddr() net.Addr {
	if p.local != nil {
		return p.local
	}
	return relayNetAddr(relay.Address{Host: "0.0.0.0"})
}

func (p *blackholePacketConn) SetDeadline(time.Time) error {
	return nil
}

func (p *blackholePacketConn) SetReadDeadline(time.Time) error {
	return nil
}

func (p *blackholePacketConn) SetWriteDeadline(time.Time) error {
	return nil
}
