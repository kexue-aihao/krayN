package engine

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"kray/pkg/kless"
	"kray/pkg/kless/transport/grpc"
	"kray/pkg/kless/transport/httpstream"
	"kray/pkg/kless/transport/httpupgrade"
	tlstransport "kray/pkg/kless/transport/tls"
	"kray/pkg/kless/transport/websocket"
	"kray/pkg/kless/transport/xhttp"
	"krayn/core/internal/config"
	"krayn/core/internal/proxy"
	"krayn/core/internal/stats"
	"krayn/core/internal/tunbridge"
)

type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
)

const (
	defaultLatencyURL     = "https://www.google.com/generate_204"
	defaultSpeedURL       = "https://speed.cloudflare.com/__down?bytes=10000000"
	egressInfoURL         = "https://api.ip.sb/geoip"
	egressInfoFallbackURL = "https://ipinfo.io/json"
	purityBaseURL         = "http://ping0.co/"

	diagnosticConnectTimeout = 8 * time.Second
	diagnosticHTTPTimeout    = 15 * time.Second
	diagnosticSpeedTimeout   = 45 * time.Second
	diagnosticRTTSamples     = 10
	maxSpeedDownloadBytes    = 100 << 20
	udpTypeUnsupported       = "unsupported"
)

type Engine struct {
	configPath string
	logger     *slog.Logger
	stats      *stats.Collector

	mu        sync.RWMutex
	cfg       config.AppConfig
	status    Status
	lastError string
	cancel    context.CancelFunc
	socks     *proxy.SOCKSServer
	tun       *tunbridge.Service
	startedAt time.Time
}

type RuntimeState struct {
	Status          Status         `json:"status"`
	LastError       string         `json:"last_error,omitempty"`
	StartedAt       time.Time      `json:"started_at,omitempty"`
	ActiveProfileID string         `json:"active_profile_id"`
	SOCKSAddress    string         `json:"socks_address"`
	APIAddress      string         `json:"api_address"`
	Mode            string         `json:"mode"`
	SystemProxyMode string         `json:"system_proxy_mode"`
	TunEnabled      bool           `json:"tun_enabled"`
	TunInterface    string         `json:"tun_interface,omitempty"`
	Stats           stats.Snapshot `json:"stats"`
}

type DiagnosticRequest struct {
	LatencyURL string `json:"latency_url,omitempty"`
	SpeedURL   string `json:"speed_url,omitempty"`
}

type DiagnosticResult struct {
	ProfileID         string   `json:"profile_id"`
	ProfileName       string   `json:"profile_name"`
	RTTMs             int64    `json:"rtt_ms,omitempty"`
	RTTSamplesMs      []int    `json:"rtt_samples_ms,omitempty"`
	RTTMaxMs          int64    `json:"rtt_max_ms,omitempty"`
	RTTStdDevMs       int64    `json:"rtt_stddev_ms,omitempty"`
	JitterMs          int64    `json:"jitter_ms,omitempty"`
	PacketLossPercent float64  `json:"packet_loss_percent,omitempty"`
	UDPType           string   `json:"udp_type,omitempty"`
	HTTPSMs           int64    `json:"https_ms,omitempty"`
	SpeedMbps         float64  `json:"speed_mbps,omitempty"`
	DownloadedBytes   int64    `json:"downloaded_bytes,omitempty"`
	EgressIP          string   `json:"egress_ip,omitempty"`
	ASN               int      `json:"asn,omitempty"`
	ASNOrganization   string   `json:"asn_organization,omitempty"`
	ISP               string   `json:"isp,omitempty"`
	Country           string   `json:"country,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	City              string   `json:"city,omitempty"`
	PuritySummary     string   `json:"purity_summary,omitempty"`
	PurityURL         string   `json:"purity_url,omitempty"`
	ResolvedIPs       []string `json:"resolved_ips,omitempty"`
	LatencyURL        string   `json:"latency_url,omitempty"`
	SpeedURL          string   `json:"speed_url,omitempty"`
	Errors            []string `json:"errors,omitempty"`
}

func New(configPath string, logger *slog.Logger) (*Engine, error) {
	if logger == nil {
		logger = slog.Default()
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	if err := config.Validate(cfg); err != nil && len(cfg.Profiles) > 0 {
		return nil, err
	}
	return &Engine{
		configPath: configPath,
		logger:     logger,
		stats:      stats.NewCollector(),
		cfg:        cfg,
		status:     StatusStopped,
	}, nil
}

func (e *Engine) Config() config.AppConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return cloneConfig(e.cfg)
}

func (e *Engine) State() RuntimeState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state := RuntimeState{
		Status:          e.status,
		LastError:       e.lastError,
		StartedAt:       e.startedAt,
		ActiveProfileID: e.cfg.ActiveProfileID,
		SOCKSAddress:    e.cfg.Local.SOCKSAddress,
		APIAddress:      e.cfg.Local.APIAddress,
		Mode:            e.cfg.Local.Mode,
		SystemProxyMode: e.cfg.Local.SystemProxyMode,
		TunEnabled:      e.cfg.Local.Tun.Enabled,
		Stats:           e.stats.Snapshot(),
	}
	if e.tun != nil {
		state.TunInterface = e.tun.InterfaceName()
	}
	return state
}

func (e *Engine) Start(parent context.Context) error {
	e.mu.Lock()
	if e.status == StatusRunning || e.status == StatusStarting {
		e.mu.Unlock()
		return nil
	}
	cfg := cloneConfig(e.cfg)
	profile, ok := config.FindProfile(cfg, cfg.ActiveProfileID)
	if !ok {
		e.status = StatusError
		e.lastError = "no active profile selected"
		e.mu.Unlock()
		return errors.New("no active profile selected")
	}
	if err := config.ValidateProfile(profile); err != nil {
		e.status = StatusError
		e.lastError = err.Error()
		e.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	socks := &proxy.SOCKSServer{
		Address: cfg.Local.SOCKSAddress,
		Dialer:  e,
		Stats:   e.stats,
		Logger:  e.logger,
	}
	e.cancel = cancel
	e.socks = socks
	e.tun = nil
	e.status = StatusStarting
	e.lastError = ""
	e.startedAt = time.Now().UTC()
	e.mu.Unlock()

	var tunService *tunbridge.Service
	if cfg.Local.Tun.Enabled {
		endpointIPs := resolveTunEndpointIPs(context.Background(), cfg.Local, profile, e.logger)
		dialer := tunbridgeDialer(func(ctx context.Context) (io.ReadWriteCloser, error) {
			return e.dialKLESSSession(ctx, profile, cfg.Local)
		})
		var err error
		tunService, err = tunbridge.Start(ctx, tunbridge.Option{
			Config:      cfg.Local.Tun,
			Dialer:      dialer,
			EndpointIPs: endpointIPs,
			Logger:      e.logger,
		})
		if err != nil {
			cancel()
			_ = socks.Close()
			e.mu.Lock()
			e.status = StatusError
			e.lastError = err.Error()
			e.cancel = nil
			e.socks = nil
			e.tun = nil
			e.mu.Unlock()
			return err
		}
		e.mu.Lock()
		e.tun = tunService
		e.mu.Unlock()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- socks.Start(ctx)
	}()
	select {
	case err := <-errCh:
		cancel()
		if tunService != nil {
			tunService.Close()
		}
		e.mu.Lock()
		e.status = StatusError
		e.lastError = err.Error()
		e.cancel = nil
		e.socks = nil
		e.tun = nil
		e.mu.Unlock()
		return err
	case <-time.After(150 * time.Millisecond):
		e.mu.Lock()
		e.status = StatusRunning
		e.mu.Unlock()
		go e.watchSOCKS(errCh)
		return nil
	}
}

func (e *Engine) Stop() error {
	e.mu.Lock()
	cancel := e.cancel
	socks := e.socks
	tun := e.tun
	e.cancel = nil
	e.socks = nil
	e.tun = nil
	e.status = StatusStopped
	e.lastError = ""
	e.startedAt = time.Time{}
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if socks != nil {
		if err := socks.Close(); err != nil {
			return err
		}
	}
	if tun != nil {
		tun.Close()
	}
	return nil
}

func (e *Engine) UpsertProfile(profile config.Profile) (config.Profile, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	profile.Transport = config.NormalizeTransport(profile.Transport)
	if err := config.ValidateProfile(profile); err != nil {
		return config.Profile{}, err
	}
	next := cloneConfig(e.cfg)
	profile = config.UpsertProfile(&next, profile)
	if err := config.Validate(next); err != nil {
		return config.Profile{}, err
	}
	if err := config.Save(e.configPath, next); err != nil {
		return config.Profile{}, err
	}
	e.cfg = next
	return profile, nil
}

func (e *Engine) DeleteProfile(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := cloneConfig(e.cfg)
	if !config.DeleteProfile(&next, id) {
		return errors.New("profile not found")
	}
	if err := config.Validate(next); err != nil {
		return err
	}
	if err := config.Save(e.configPath, next); err != nil {
		return err
	}
	e.cfg = next
	return nil
}

func (e *Engine) SetActiveProfile(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := cloneConfig(e.cfg)
	if _, ok := config.FindProfile(next, id); !ok {
		return errors.New("profile not found")
	}
	next.ActiveProfileID = id
	if err := config.Save(e.configPath, next); err != nil {
		return err
	}
	e.cfg = next
	return nil
}

func (e *Engine) UpdateLocal(local config.LocalConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := cloneConfig(e.cfg)
	next.Local = local
	config.Normalize(&next)
	if err := config.Validate(next); err != nil {
		return err
	}
	if err := config.Save(e.configPath, next); err != nil {
		return err
	}
	e.cfg = next
	return nil
}

func (e *Engine) DialContext(ctx context.Context, target proxy.Target) (io.ReadWriteCloser, error) {
	e.mu.RLock()
	cfg := cloneConfig(e.cfg)
	e.mu.RUnlock()
	if config.NormalizeMode(cfg.Local.Mode) == "direct" {
		resolver := newResolver(cfg.Local)
		return resolver.DialContext(ctx, "tcp", target.Address())
	}
	profile, ok := config.FindProfile(cfg, cfg.ActiveProfileID)
	if !ok {
		return nil, errors.New("no active profile selected")
	}
	return e.dialProfileContext(ctx, profile, target, cfg.Local)
}

func (e *Engine) TestProfile(ctx context.Context, profileID string, req DiagnosticRequest) (DiagnosticResult, error) {
	cfg := e.Config()
	if profileID == "" {
		profileID = cfg.ActiveProfileID
	}
	profile, ok := config.FindProfile(cfg, profileID)
	if !ok {
		return DiagnosticResult{}, errors.New("profile not found")
	}
	if err := config.ValidateProfile(profile); err != nil {
		return DiagnosticResult{}, err
	}
	result := DiagnosticResult{
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
		LatencyURL:  normalizedTestURL(req.LatencyURL, defaultLatencyURL),
		SpeedURL:    normalizedTestURL(req.SpeedURL, defaultSpeedURL),
		UDPType:     udpTypeUnsupported,
	}

	resolver := newResolver(cfg.Local)
	endpoint, err := endpointAddress(profile)
	if err != nil {
		result.Errors = append(result.Errors, "endpoint: "+err.Error())
	} else {
		resolveCtx, cancel := context.WithTimeout(ctx, diagnosticConnectTimeout)
		if ips, err := resolver.LookupIP(resolveCtx, endpoint.host); err != nil {
			result.Errors = append(result.Errors, "resolve: "+err.Error())
		} else {
			result.ResolvedIPs = ipStrings(ips)
		}
		cancel()

		stats, err := measureRTT(ctx, resolver, endpoint.address, diagnosticRTTSamples, diagnosticConnectTimeout)
		result.RTTMs = stats.averageMs
		result.RTTSamplesMs = stats.samples
		result.RTTMaxMs = stats.maxMs
		result.RTTStdDevMs = stats.stdDevMs
		result.JitterMs = stats.stdDevMs
		result.PacketLossPercent = stats.packetLossPercent
		if err != nil {
			result.Errors = append(result.Errors, "rtt: "+err.Error())
		}
	}

	latencyClient := e.httpClientViaProfile(profile, cfg.Local, diagnosticHTTPTimeout)
	if latencyURL := result.LatencyURL; latencyURL != "" {
		start := time.Now()
		if err := requestSmall(ctx, latencyClient, latencyURL); err != nil {
			result.Errors = append(result.Errors, "https_latency: "+err.Error())
		} else {
			result.HTTPSMs = millisSince(start)
		}
	}

	egressClient := e.httpClientViaProfile(profile, cfg.Local, diagnosticHTTPTimeout)
	if info, err := fetchEgressInfo(ctx, egressClient); err != nil {
		result.Errors = append(result.Errors, "egress_ip: "+err.Error())
	} else {
		result.EgressIP = info.IP
		result.ASN = info.ASN
		result.ASNOrganization = info.ASNOrganization
		result.ISP = info.ISP
		result.Country = info.Country
		result.CountryCode = info.CountryCode
		result.City = info.City
		if info.IP != "" {
			result.PurityURL = purityBaseURL + "?ip=" + url.QueryEscape(info.IP)
			if purity, err := fetchPuritySummary(ctx, egressClient, result.PurityURL); err != nil {
				result.Errors = append(result.Errors, "ip_purity: "+err.Error())
			} else {
				result.PuritySummary = purity
			}
		}
	}

	speedClient := e.httpClientViaProfile(profile, cfg.Local, diagnosticSpeedTimeout)
	if speedURL := result.SpeedURL; speedURL != "" {
		downloaded, elapsed, err := measureDownload(ctx, speedClient, speedURL)
		if err != nil {
			result.Errors = append(result.Errors, "speed: "+err.Error())
		} else {
			result.DownloadedBytes = downloaded
			if elapsed > 0 {
				result.SpeedMbps = float64(downloaded*8) / elapsed.Seconds() / 1_000_000
			}
		}
	}
	return result, nil
}

func (e *Engine) dialProfileContext(ctx context.Context, profile config.Profile, target proxy.Target, local config.LocalConfig) (io.ReadWriteCloser, error) {
	secure, err := e.dialKLESSSession(ctx, profile, local)
	if err != nil {
		return nil, err
	}
	if err := proxy.WriteConnectRequest(secure, target); err != nil {
		_ = secure.Close()
		return nil, fmt.Errorf("relay connect request: %w", err)
	}
	if err := proxy.ReadConnectResponse(secure); err != nil {
		_ = secure.Close()
		return nil, fmt.Errorf("relay connect response: %w", err)
	}
	return secure, nil
}

func (e *Engine) dialKLESSSession(ctx context.Context, profile config.Profile, local config.LocalConfig) (io.ReadWriteCloser, error) {
	raw, err := dialTransport(ctx, profile, local)
	if err != nil {
		return nil, fmt.Errorf("dial kless transport: %w", err)
	}
	secure, err := e.clientHandshake(raw, profile)
	if err != nil {
		_ = raw.Close()
		if isKlessHandshakeTruncated(err) {
			return nil, fmt.Errorf("kless handshake: server closed the connection before handshake completed; check client_id, client_secret, server_public_key/server_signing_key, transport, and server relay mode. %s: %w", klessHandshakeEOFHint(), err)
		}
		return nil, fmt.Errorf("kless handshake: %w", err)
	}
	return secure, nil
}

type tunbridgeDialer func(context.Context) (io.ReadWriteCloser, error)

func (f tunbridgeDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return f(ctx)
}

func (e *Engine) clientHandshake(raw io.ReadWriteCloser, profile config.Profile) (*kless.Conn, error) {
	secret, err := decodeKLESSKey(profile.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("decode client secret: %w", err)
	}
	publicKey, err := decodeKLESSKey(profile.ServerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode server public key: %w", err)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("server public key must be %d bytes", ed25519.PublicKeySize)
	}
	return kless.ClientHandshake(raw, kless.ClientConfig{
		ClientID:         profile.ClientID,
		ClientSecret:     secret,
		ServerSigningKey: ed25519.PublicKey(publicKey),
		Capabilities:     kless.CapabilityAll,
		PaddingMin:       profile.HandshakePadding.Min,
		PaddingMax:       profile.HandshakePadding.Max,
	})
}

func klessHandshakeEOFHint() string {
	return "If the remote node is managed by Knode, the client endpoint must be a public inbound in mode \"kless-server\" (often named \"public-kless\"). A plain \"tcp\" / \"local-tcp\" inbound is only a forwarding port and cannot accept a direct krayN client."
}

func isKlessHandshakeTruncated(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, needle := range []string{
		"connection was aborted",
		"connection reset",
		"forcibly closed",
		"broken pipe",
	} {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}

func decodeKLESSKey(text string) ([]byte, error) {
	normalized := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		default:
			return r
		}
	}, strings.TrimSpace(text))
	if normalized == "" {
		return nil, base64.CorruptInputError(0)
	}
	encodings := []*base64.Encoding{
		base64.RawStdEncoding,
		base64.StdEncoding,
		base64.RawURLEncoding,
		base64.URLEncoding,
	}
	var lastErr error
	for _, encoding := range encodings {
		key, err := encoding.DecodeString(normalized)
		if err == nil {
			return key, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

type ipResolver interface {
	LookupIP(ctx context.Context, host string) ([]net.IP, error)
}

func resolveProxyTarget(ctx context.Context, target proxy.Target, lookup ipResolver) (proxy.Target, error) {
	if target.Host == "" || net.ParseIP(target.Host) != nil || lookup == nil {
		return target, nil
	}
	ips, err := lookup.LookupIP(ctx, target.Host)
	if err != nil || len(ips) == 0 {
		return target, err
	}
	chosen := choosePreferredIP(ips)
	if chosen == nil {
		return target, nil
	}
	return proxy.Target{Host: chosen.String(), Port: target.Port}, nil
}

func choosePreferredIP(ips []net.IP) net.IP {
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			return ip4
		}
	}
	for _, ip := range ips {
		if ip != nil {
			return ip
		}
	}
	return nil
}

func (e *Engine) watchSOCKS(errCh <-chan error) {
	err := <-errCh
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.status == StatusStopped {
		return
	}
	if err != nil {
		e.status = StatusError
		e.lastError = err.Error()
		e.logger.Error("socks server stopped", "error", err)
		return
	}
	e.status = StatusStopped
}

func resolveTunEndpointIPs(ctx context.Context, local config.LocalConfig, profile config.Profile, logger *slog.Logger) []net.IP {
	endpoint, err := endpointAddress(profile)
	if err != nil {
		if logger != nil {
			logger.Warn("resolve tun endpoint failed", "error", err)
		}
		return nil
	}
	host := endpoint.host
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}
	}
	resolveCtx, cancel := context.WithTimeout(ctx, diagnosticConnectTimeout)
	defer cancel()
	ips, err := newResolver(local).LookupIP(resolveCtx, host)
	if err != nil && logger != nil {
		logger.Warn("resolve tun endpoint failed", "host", host, "error", err)
	}
	return ips
}

func dialTransport(ctx context.Context, profile config.Profile, local config.LocalConfig) (io.ReadWriteCloser, error) {
	resolver := newResolver(local)
	switch config.NormalizeTransport(profile.Transport) {
	case "tcp":
		return resolver.DialContext(ctx, "tcp", profile.Endpoint)
	case "tls":
		cfg := tlsConfigForEndpoint(profile, profile.Endpoint, true)
		conn, err := resolver.DialContext(ctx, "tcp", profile.Endpoint)
		if err != nil {
			return nil, err
		}
		tlsConn := tls.Client(conn, cfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return tlsConn, nil
	case "websocket":
		return dialWebSocketTransport(ctx, resolver, profile)
	case "http-upgrade":
		return dialHTTPUpgradeTransport(ctx, resolver, profile)
	case "http-stream":
		return httpstream.Dial(ctx, httpClient(profile, local), profile.Endpoint, headers(profile.Headers))
	case "grpc":
		return grpc.Dial(ctx, httpClient(profile, local), profile.Endpoint, headers(profile.Headers))
	case "xhttp":
		return xhttp.Dial(ctx, httpClient(profile, local), profile.Endpoint, headers(profile.Headers))
	default:
		return nil, fmt.Errorf("unsupported transport %q", profile.Transport)
	}
}

func headers(in map[string]string) http.Header {
	out := make(http.Header, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}
		out.Set(key, value)
	}
	return out
}

func httpClient(profile config.Profile, local config.LocalConfig) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	resolver := newResolver(local)
	transport.DialContext = resolver.DialContext
	if tlsCfg := tlsConfigForURL(profile, profile.Endpoint, []string{"h2", "http/1.1"}); tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	return &http.Client{Transport: transport}
}

func dialWebSocketTransport(ctx context.Context, resolver resolver, profile config.Profile) (io.ReadWriteCloser, error) {
	parsed, err := url.Parse(profile.Endpoint)
	if err != nil {
		return nil, err
	}
	conn, err := dialURLConn(ctx, resolver, parsed, tlsConfigForURL(profile, profile.Endpoint, []string{"http/1.1"}), "ws", "wss")
	if err != nil {
		return nil, err
	}
	key, err := randomWebSocketKey()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profile.Endpoint, nil)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", key)
	for name, values := range headers(profile.Headers) {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		return nil, fmt.Errorf("unexpected websocket status: %s", resp.Status)
	}
	if resp.Header.Get("Sec-WebSocket-Accept") != webSocketAcceptKey(key) {
		_ = conn.Close()
		return nil, errors.New("invalid websocket accept key")
	}
	return websocket.NewConn(&bufferedNetConn{Conn: conn, Reader: reader}, true), nil
}

func dialHTTPUpgradeTransport(ctx context.Context, resolver resolver, profile config.Profile) (io.ReadWriteCloser, error) {
	parsed, err := url.Parse(profile.Endpoint)
	if err != nil {
		return nil, err
	}
	conn, err := dialURLConn(ctx, resolver, parsed, tlsConfigForURL(profile, profile.Endpoint, []string{"http/1.1"}), "http", "https")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profile.Endpoint, nil)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", httpupgrade.DefaultUpgradeName)
	for name, values := range headers(profile.Headers) {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		return nil, fmt.Errorf("unexpected upgrade status: %s", resp.Status)
	}
	if !strings.EqualFold(resp.Header.Get("Upgrade"), httpupgrade.DefaultUpgradeName) {
		_ = conn.Close()
		return nil, errors.New("server did not select kless upgrade")
	}
	return &bufferedNetConn{Conn: conn, Reader: reader}, nil
}

func dialURLConn(ctx context.Context, resolver resolver, parsed *url.URL, tlsCfg *tls.Config, plainScheme, tlsScheme string) (net.Conn, error) {
	address := urlHostPort(parsed)
	switch parsed.Scheme {
	case plainScheme:
		return resolver.DialContext(ctx, "tcp", address)
	case tlsScheme:
		conn, err := resolver.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		if tlsCfg == nil {
			tlsCfg = &tls.Config{
				MinVersion: tls.VersionTLS13,
				ServerName: parsed.Hostname(),
				NextProtos: []string{"http/1.1"},
			}
		}
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return tlsConn, nil
	default:
		return nil, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
}

func urlHostPort(u *url.URL) string {
	if u.Port() != "" {
		return u.Host
	}
	switch u.Scheme {
	case "https", "wss":
		return net.JoinHostPort(u.Hostname(), "443")
	default:
		return net.JoinHostPort(u.Hostname(), "80")
	}
}

func randomWebSocketKey() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b[:]), nil
}

func webSocketAcceptKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

type bufferedNetConn struct {
	net.Conn
	Reader *bufio.Reader
}

func (c *bufferedNetConn) Read(p []byte) (int, error) {
	if c.Reader != nil && c.Reader.Buffered() > 0 {
		return c.Reader.Read(p)
	}
	return c.Conn.Read(p)
}

func (e *Engine) httpClientViaProfile(profile config.Profile, local config.LocalConfig, timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		target, err := proxy.ParseTarget(address)
		if err != nil {
			return nil, err
		}
		rwc, err := e.dialProfileContext(ctx, profile, target, local)
		if err != nil {
			return nil, err
		}
		conn, ok := rwc.(net.Conn)
		if !ok {
			return &netConnAdapter{ReadWriteCloser: rwc}, nil
		}
		return conn, nil
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

type endpointParts struct {
	host    string
	port    string
	address string
}

func endpointAddress(profile config.Profile) (endpointParts, error) {
	switch config.NormalizeTransport(profile.Transport) {
	case "tcp", "tls":
		host, port, err := net.SplitHostPort(profile.Endpoint)
		if err != nil {
			return endpointParts{}, err
		}
		return endpointParts{host: host, port: port, address: net.JoinHostPort(host, port)}, nil
	default:
		parsed, err := url.Parse(profile.Endpoint)
		if err != nil {
			return endpointParts{}, err
		}
		host := parsed.Hostname()
		port := parsed.Port()
		if host == "" {
			return endpointParts{}, errors.New("endpoint host is required")
		}
		if port == "" {
			switch parsed.Scheme {
			case "https", "wss":
				port = "443"
			default:
				port = "80"
			}
		}
		return endpointParts{host: host, port: port, address: net.JoinHostPort(host, port)}, nil
	}
}

type rttStats struct {
	averageMs         int64
	samples           []int
	maxMs             int64
	stdDevMs          int64
	packetLossPercent float64
}

func measureRTT(ctx context.Context, resolver resolver, address string, sampleCount int, sampleTimeout time.Duration) (rttStats, error) {
	if sampleCount <= 0 {
		sampleCount = 1
	}
	samples := make([]int, 0, sampleCount)
	failures := 0
	var lastErr error

	for i := 0; i < sampleCount; i++ {
		if err := ctx.Err(); err != nil {
			failures += sampleCount - i
			lastErr = err
			break
		}
		sampleCtx, cancel := context.WithTimeout(ctx, sampleTimeout)
		start := time.Now()
		conn, err := resolver.DialContext(sampleCtx, "tcp", address)
		elapsed := millisSince(start)
		cancel()
		if err != nil {
			failures++
			lastErr = err
			continue
		}
		_ = conn.Close()
		samples = append(samples, int(elapsed))
	}

	stats := calculateRTTStats(samples, sampleCount, failures)
	if len(samples) == 0 {
		if lastErr != nil {
			return stats, fmt.Errorf("all %d samples failed: %w", sampleCount, lastErr)
		}
		return stats, fmt.Errorf("all %d samples failed", sampleCount)
	}

	if failures > 0 {
		if lastErr != nil {
			return stats, fmt.Errorf("%d of %d samples failed: %w", failures, sampleCount, lastErr)
		}
		return stats, fmt.Errorf("%d of %d samples failed", failures, sampleCount)
	}
	return stats, nil
}

func calculateRTTStats(samples []int, sampleCount, failures int) rttStats {
	stats := rttStats{
		samples:           append([]int(nil), samples...),
		packetLossPercent: float64(failures) * 100 / float64(sampleCount),
	}
	if len(samples) == 0 {
		return stats
	}

	var sum int64
	for _, sample := range samples {
		value := int64(sample)
		sum += value
		if value > stats.maxMs {
			stats.maxMs = value
		}
	}
	average := float64(sum) / float64(len(samples))
	stats.averageMs = int64(math.Round(average))

	var variance float64
	for _, sample := range samples {
		diff := float64(sample) - average
		variance += diff * diff
	}
	stats.stdDevMs = int64(math.Round(math.Sqrt(variance / float64(len(samples)))))
	return stats
}

func normalizedTestURL(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = fallback
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fallback
	}
	return trimmed
}

func millisSince(start time.Time) int64 {
	elapsed := time.Since(start)
	if elapsed <= 0 {
		return 1
	}
	ms := elapsed.Milliseconds()
	if ms <= 0 {
		return 1
	}
	return ms
}

func ipStrings(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		text := ip.String()
		if !slices.Contains(out, text) {
			out = append(out, text)
		}
	}
	return out
}

func requestSmall(ctx context.Context, client *http.Client, rawURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "krayN/diagnostics")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 256*1024)
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	return nil
}

func measureDownload(ctx context.Context, client *http.Client, rawURL string) (int64, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "krayN/diagnostics")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return 0, 0, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	downloaded, err := io.Copy(io.Discard, io.LimitReader(resp.Body, maxSpeedDownloadBytes))
	elapsed := time.Since(start)
	if err != nil {
		return downloaded, elapsed, err
	}
	return downloaded, elapsed, nil
}

type egressInfo struct {
	IP              string
	ASN             int
	ASNOrganization string
	ISP             string
	Country         string
	CountryCode     string
	City            string
}

func fetchEgressInfo(ctx context.Context, client *http.Client) (egressInfo, error) {
	info, err := fetchIPSB(ctx, client, egressInfoURL)
	if err == nil && info.IP != "" {
		return info, nil
	}
	fallback, fallbackErr := fetchIPInfo(ctx, client, egressInfoFallbackURL)
	if fallbackErr == nil && fallback.IP != "" {
		return fallback, nil
	}
	if err != nil {
		return egressInfo{}, err
	}
	return egressInfo{}, fallbackErr
}

func fetchIPSB(ctx context.Context, client *http.Client, rawURL string) (egressInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return egressInfo{}, err
	}
	req.Header.Set("User-Agent", "krayN/diagnostics")
	resp, err := client.Do(req)
	if err != nil {
		return egressInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return egressInfo{}, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	var payload struct {
		IP              string `json:"ip"`
		ASN             int    `json:"asn"`
		ASNOrganization string `json:"asn_organization"`
		Organization    string `json:"organization"`
		ISP             string `json:"isp"`
		Country         string `json:"country"`
		CountryCode     string `json:"country_code"`
		City            string `json:"city"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return egressInfo{}, err
	}
	return egressInfo{
		IP:              payload.IP,
		ASN:             payload.ASN,
		ASNOrganization: firstNonEmpty(payload.ASNOrganization, payload.Organization),
		ISP:             firstNonEmpty(payload.ISP, payload.Organization),
		Country:         payload.Country,
		CountryCode:     payload.CountryCode,
		City:            payload.City,
	}, nil
}

func fetchIPInfo(ctx context.Context, client *http.Client, rawURL string) (egressInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return egressInfo{}, err
	}
	req.Header.Set("User-Agent", "krayN/diagnostics")
	resp, err := client.Do(req)
	if err != nil {
		return egressInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return egressInfo{}, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	var payload struct {
		IP      string `json:"ip"`
		Org     string `json:"org"`
		Country string `json:"country"`
		City    string `json:"city"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return egressInfo{}, err
	}
	asn, org := parseIPInfoOrg(payload.Org)
	return egressInfo{
		IP:              payload.IP,
		ASN:             asn,
		ASNOrganization: org,
		ISP:             org,
		CountryCode:     payload.Country,
		City:            payload.City,
	}, nil
}

func fetchPuritySummary(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "krayN/diagnostics")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	summary := extractPuritySummary(string(body))
	if summary == "" {
		return "", errors.New("ping0 summary not found")
	}
	return summary, nil
}

func extractPuritySummary(html string) string {
	parts := []string{}
	if risk := extractRiskText(html); risk != "" {
		parts = append(parts, "Ping0 "+risk)
	}
	if native := extractLineLabels(html, "line-nativeip"); len(native) > 0 {
		parts = append(parts, strings.Join(native, " / "))
	}
	if ipTypes := extractLineLabels(html, "line-iptype"); len(ipTypes) > 0 {
		parts = append(parts, strings.Join(ipTypes, " / "))
	}
	return strings.Join(parts, " | ")
}

func extractRiskText(html string) string {
	segment := extractSegment(html, "line-risk")
	if segment == "" {
		return ""
	}
	current := segment
	if idx := strings.Index(current, "riskcurrent"); idx >= 0 {
		current = current[idx:]
	}
	value := extractBetween(current, `<span class="value">`, "</span>")
	label := strings.TrimSpace(extractBetween(current, `<span class="lab">`, "</span>"))
	switch {
	case value != "" && label != "":
		return value + " " + label
	case value != "":
		return value
	case label != "":
		return label
	default:
		title := extractBetween(current, `title="`, `"`)
		if fields := strings.Fields(title); len(fields) > 0 {
			return strings.Join(fields, " ")
		}
		return ""
	}
}

func extractLineLabels(html, className string) []string {
	segment := extractSegment(html, className)
	if segment == "" {
		return nil
	}
	labels := []string{}
	remaining := segment
	for {
		start := strings.Index(remaining, "<span")
		if start < 0 {
			break
		}
		remaining = remaining[start:]
		tagEnd := strings.Index(remaining, ">")
		if tagEnd < 0 {
			break
		}
		tag := remaining[:tagEnd+1]
		if strings.Contains(tag, "label") {
			end := strings.Index(remaining[tagEnd+1:], "</span>")
			if end < 0 {
				break
			}
			text := cleanHTMLText(remaining[tagEnd+1 : tagEnd+1+end])
			if text != "" && !slices.Contains(labels, text) {
				labels = append(labels, text)
			}
			remaining = remaining[tagEnd+1+end+len("</span>"):]
			continue
		}
		remaining = remaining[tagEnd+1:]
	}
	return labels
}

func extractSegment(html, className string) string {
	idx := strings.Index(html, className)
	if idx < 0 {
		return ""
	}
	end := strings.Index(html[idx+len(className):], `<div class="line `)
	if end < 0 {
		end = min(len(html)-idx, 3000)
	} else {
		end += len(className)
	}
	return html[idx : idx+end]
}

func extractBetween(text, startMarker, endMarker string) string {
	start := strings.Index(text, startMarker)
	if start < 0 {
		return ""
	}
	start += len(startMarker)
	end := strings.Index(text[start:], endMarker)
	if end < 0 {
		return ""
	}
	return cleanHTMLText(text[start : start+end])
}

func cleanHTMLText(text string) string {
	var out strings.Builder
	inTag := false
	for _, r := range text {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out.WriteRune(r)
			}
		}
	}
	cleaned := strings.NewReplacer(
		"&nbsp;", " ",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&#39;", "'",
		"&quot;", `"`,
	).Replace(out.String())
	return strings.Join(strings.Fields(cleaned), " ")
}

func parseIPInfoOrg(org string) (int, string) {
	fields := strings.Fields(org)
	if len(fields) == 0 {
		return 0, ""
	}
	if strings.HasPrefix(strings.ToUpper(fields[0]), "AS") {
		var asn int
		for _, ch := range fields[0][2:] {
			if ch < '0' || ch > '9' {
				asn = 0
				break
			}
			asn = asn*10 + int(ch-'0')
		}
		return asn, strings.TrimSpace(strings.TrimPrefix(org, fields[0]))
	}
	return 0, org
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type netConnAdapter struct {
	io.ReadWriteCloser
}

func (c *netConnAdapter) LocalAddr() net.Addr              { return dummyAddr("krayn-local") }
func (c *netConnAdapter) RemoteAddr() net.Addr             { return dummyAddr("krayn-remote") }
func (c *netConnAdapter) SetDeadline(time.Time) error      { return nil }
func (c *netConnAdapter) SetReadDeadline(time.Time) error  { return nil }
func (c *netConnAdapter) SetWriteDeadline(time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return "krayn" }
func (a dummyAddr) String() string  { return string(a) }

type resolver struct {
	local config.LocalConfig
}

func newResolver(local config.LocalConfig) resolver {
	local.ResolverType = config.NormalizeResolverType(local.ResolverType)
	local.ResolverAddress = strings.TrimSpace(local.ResolverAddress)
	return resolver{local: local}
}

func (r resolver) netDialer() *net.Dialer {
	return &net.Dialer{Timeout: diagnosticConnectTimeout}
}

func (r resolver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || net.ParseIP(host) != nil || config.NormalizeResolverType(r.local.ResolverType) == "system" {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, address)
	}
	ips, err := r.LookupIP(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	var dialer net.Dialer
	for _, ip := range ips {
		if network == "tcp4" && ip.To4() == nil {
			continue
		}
		if network == "tcp6" && (ip.To16() == nil || ip.To4() != nil) {
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no usable IP address for %s", host)
}

func (r resolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	switch config.NormalizeResolverType(r.local.ResolverType) {
	case "dns":
		return r.lookupIPViaDNS(ctx, host)
	case "doh":
		return r.lookupIPViaDoH(ctx, host)
	default:
		resolver := net.DefaultResolver
		addrs, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		ips := make([]net.IP, 0, len(addrs))
		for _, addr := range addrs {
			ips = append(ips, addr.IP)
		}
		return dedupeIPs(ips), nil
	}
}

func (r resolver) lookupIPViaDNS(ctx context.Context, host string) ([]net.IP, error) {
	address := strings.TrimSpace(r.local.ResolverAddress)
	if address == "" {
		return nil, errors.New("dns resolver address is empty")
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "53")
	}
	dialer := &net.Dialer{}
	netResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "udp", address)
		},
	}
	addrs, err := netResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}
	return dedupeIPs(ips), nil
}

func (r resolver) lookupIPViaDoH(ctx context.Context, host string) ([]net.IP, error) {
	endpoint := strings.TrimSpace(r.local.ResolverAddress)
	if endpoint == "" {
		return nil, errors.New("doh resolver url is empty")
	}
	a, errA := queryDoH(ctx, endpoint, host, dnsTypeA)
	aaaa, errAAAA := queryDoH(ctx, endpoint, host, dnsTypeAAAA)
	ips := dedupeIPs(append(a, aaaa...))
	if len(ips) > 0 {
		return ips, nil
	}
	if errA != nil {
		return nil, errA
	}
	if errAAAA != nil {
		return nil, errAAAA
	}
	return nil, fmt.Errorf("no IP address found for %s", host)
}

func dedupeIPs(ips []net.IP) []net.IP {
	out := make([]net.IP, 0, len(ips))
	seen := map[string]struct{}{}
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		text := ip.String()
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, ip)
	}
	return out
}

const (
	dnsTypeA    uint16 = 1
	dnsTypeAAAA uint16 = 28
)

func queryDoH(ctx context.Context, endpoint, host string, qtype uint16) ([]net.IP, error) {
	if ips, err := queryDoHJSON(ctx, endpoint, host, qtype); err == nil && len(ips) > 0 {
		return ips, nil
	}
	return queryDoHWire(ctx, endpoint, host, qtype)
}

func queryDoHJSON(ctx context.Context, endpoint, host string, qtype uint16) ([]net.IP, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set("name", host)
	query.Set("type", dnsTypeName(qtype))
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("doh json status: %s", resp.Status)
	}
	var payload struct {
		Status int `json:"Status"`
		Answer []struct {
			Type uint16 `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Status != 0 {
		return nil, fmt.Errorf("doh json status code %d", payload.Status)
	}
	ips := make([]net.IP, 0, len(payload.Answer))
	for _, answer := range payload.Answer {
		if answer.Type != qtype {
			continue
		}
		if ip := net.ParseIP(answer.Data); ip != nil {
			ips = append(ips, ip)
		}
	}
	return dedupeIPs(ips), nil
}

func queryDoHWire(ctx context.Context, endpoint, host string, qtype uint16) ([]net.IP, error) {
	raw, err := buildDNSQuery(host, qtype)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-message")
	req.Header.Set("Content-Type", "application/dns-message")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("doh status: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	return parseDNSResponse(body, qtype)
}

func dnsTypeName(qtype uint16) string {
	switch qtype {
	case dnsTypeAAAA:
		return "AAAA"
	default:
		return "A"
	}
}

func buildDNSQuery(host string, qtype uint16) ([]byte, error) {
	var id [2]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	buf := make([]byte, 12, 512)
	copy(buf[0:2], id[:])
	buf[2] = 0x01
	buf[5] = 0x01
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if label == "" || len(label) > 63 {
			return nil, fmt.Errorf("invalid dns label %q", label)
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0)
	var tail [4]byte
	binary.BigEndian.PutUint16(tail[0:2], qtype)
	binary.BigEndian.PutUint16(tail[2:4], 1)
	buf = append(buf, tail[:]...)
	return buf, nil
}

func parseDNSResponse(body []byte, qtype uint16) ([]net.IP, error) {
	if len(body) < 12 {
		return nil, errors.New("short dns response")
	}
	qdCount := int(binary.BigEndian.Uint16(body[4:6]))
	anCount := int(binary.BigEndian.Uint16(body[6:8]))
	offset := 12
	var err error
	for i := 0; i < qdCount; i++ {
		offset, err = skipDNSName(body, offset)
		if err != nil {
			return nil, err
		}
		offset += 4
		if offset > len(body) {
			return nil, errors.New("truncated dns question")
		}
	}
	ips := make([]net.IP, 0, anCount)
	for i := 0; i < anCount; i++ {
		offset, err = skipDNSName(body, offset)
		if err != nil {
			return nil, err
		}
		if offset+10 > len(body) {
			return nil, errors.New("truncated dns answer")
		}
		answerType := binary.BigEndian.Uint16(body[offset : offset+2])
		answerClass := binary.BigEndian.Uint16(body[offset+2 : offset+4])
		rdLength := int(binary.BigEndian.Uint16(body[offset+8 : offset+10]))
		offset += 10
		if offset+rdLength > len(body) {
			return nil, errors.New("truncated dns rdata")
		}
		if answerClass == 1 && answerType == qtype {
			switch qtype {
			case dnsTypeA:
				if rdLength == net.IPv4len {
					ips = append(ips, net.IP(body[offset:offset+rdLength]).To4())
				}
			case dnsTypeAAAA:
				if rdLength == net.IPv6len {
					ips = append(ips, append(net.IP(nil), body[offset:offset+rdLength]...))
				}
			}
		}
		offset += rdLength
	}
	return dedupeIPs(ips), nil
}

func skipDNSName(body []byte, offset int) (int, error) {
	for {
		if offset >= len(body) {
			return 0, errors.New("truncated dns name")
		}
		length := int(body[offset])
		offset++
		if length == 0 {
			return offset, nil
		}
		if length&0xc0 == 0xc0 {
			if offset >= len(body) {
				return 0, errors.New("truncated dns pointer")
			}
			return offset + 1, nil
		}
		if length&0xc0 != 0 {
			return 0, errors.New("unsupported dns label")
		}
		offset += length
		if offset > len(body) {
			return 0, errors.New("truncated dns label")
		}
	}
}

func tlsConfigForURL(profile config.Profile, rawURL string, nextProtos []string) *tls.Config {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	if parsed.Scheme != "https" && parsed.Scheme != "wss" {
		return nil
	}
	return tlsConfig(profile, parsed.Hostname(), nextProtos)
}

func tlsConfigForEndpoint(profile config.Profile, endpoint string, useKLESSALPN bool) *tls.Config {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		host = endpoint
	}
	nextProtos := []string(nil)
	if useKLESSALPN {
		nextProtos = []string{tlstransport.ALPN}
	}
	return tlsConfig(profile, host, nextProtos)
}

func tlsConfig(profile config.Profile, host string, nextProtos []string) *tls.Config {
	serverName := profile.ServerName
	if serverName == "" {
		serverName = host
	}
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		ServerName:         serverName,
		InsecureSkipVerify: profile.SkipTLSVerify,
		NextProtos:         nextProtos,
	}
}

func cloneConfig(cfg config.AppConfig) config.AppConfig {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg
	}
	var out config.AppConfig
	if err := json.Unmarshal(raw, &out); err != nil {
		return cfg
	}
	return out
}
