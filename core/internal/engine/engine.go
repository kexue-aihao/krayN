package engine

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"kray/pkg/kless"
	"kray/pkg/kless/transport/grpc"
	"kray/pkg/kless/transport/httpstream"
	"kray/pkg/kless/transport/httpupgrade"
	"kray/pkg/kless/transport/tcp"
	tlstransport "kray/pkg/kless/transport/tls"
	"kray/pkg/kless/transport/websocket"
	"kray/pkg/kless/transport/xhttp"
	"krayn/core/internal/config"
	"krayn/core/internal/proxy"
	"krayn/core/internal/stats"
)

type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
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
	startedAt time.Time
}

type RuntimeState struct {
	Status          Status         `json:"status"`
	LastError       string         `json:"last_error,omitempty"`
	StartedAt       time.Time      `json:"started_at,omitempty"`
	ActiveProfileID string         `json:"active_profile_id"`
	SOCKSAddress    string         `json:"socks_address"`
	APIAddress      string         `json:"api_address"`
	Stats           stats.Snapshot `json:"stats"`
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
	return RuntimeState{
		Status:          e.status,
		LastError:       e.lastError,
		StartedAt:       e.startedAt,
		ActiveProfileID: e.cfg.ActiveProfileID,
		SOCKSAddress:    e.cfg.Local.SOCKSAddress,
		APIAddress:      e.cfg.Local.APIAddress,
		Stats:           e.stats.Snapshot(),
	}
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
	e.status = StatusStarting
	e.lastError = ""
	e.startedAt = time.Now().UTC()
	e.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- socks.Start(ctx)
	}()
	select {
	case err := <-errCh:
		cancel()
		e.mu.Lock()
		e.status = StatusError
		e.lastError = err.Error()
		e.cancel = nil
		e.socks = nil
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
	e.cancel = nil
	e.socks = nil
	e.status = StatusStopped
	e.lastError = ""
	e.startedAt = time.Time{}
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if socks != nil {
		return socks.Close()
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
	if err := config.Validate(next); err != nil && len(next.Profiles) > 0 {
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
	profile, ok := config.FindProfile(cfg, cfg.ActiveProfileID)
	if !ok {
		return nil, errors.New("no active profile selected")
	}
	raw, err := dialTransport(ctx, profile)
	if err != nil {
		return nil, err
	}
	secure, err := e.clientHandshake(raw, profile)
	if err != nil {
		_ = raw.Close()
		return nil, err
	}
	if err := proxy.WriteConnectRequest(secure, target); err != nil {
		_ = secure.Close()
		return nil, err
	}
	if err := proxy.ReadConnectResponse(secure); err != nil {
		_ = secure.Close()
		return nil, err
	}
	return secure, nil
}

func (e *Engine) clientHandshake(raw io.ReadWriteCloser, profile config.Profile) (*kless.Conn, error) {
	secret, err := kless.DecodeKey(profile.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("decode client secret: %w", err)
	}
	publicKey, err := kless.DecodeKey(profile.ServerPublicKey)
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

func dialTransport(ctx context.Context, profile config.Profile) (io.ReadWriteCloser, error) {
	switch config.NormalizeTransport(profile.Transport) {
	case "tcp":
		return tcp.Dial(ctx, profile.Endpoint)
	case "tls":
		return tlstransport.Dial(ctx, profile.Endpoint, tlsConfigForEndpoint(profile, profile.Endpoint, true))
	case "websocket":
		return websocket.Dial(ctx, profile.Endpoint, headers(profile.Headers), tlsConfigForURL(profile, profile.Endpoint, []string{"http/1.1"}))
	case "http-upgrade":
		return httpupgrade.Dial(ctx, profile.Endpoint, headers(profile.Headers), tlsConfigForURL(profile, profile.Endpoint, []string{"http/1.1"}))
	case "http-stream":
		return httpstream.Dial(ctx, httpClient(profile), profile.Endpoint, headers(profile.Headers))
	case "grpc":
		return grpc.Dial(ctx, httpClient(profile), profile.Endpoint, headers(profile.Headers))
	case "xhttp":
		return xhttp.Dial(ctx, httpClient(profile), profile.Endpoint, headers(profile.Headers))
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

func httpClient(profile config.Profile) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg := tlsConfigForURL(profile, profile.Endpoint, []string{"h2", "http/1.1"}); tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	return &http.Client{Transport: transport}
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
