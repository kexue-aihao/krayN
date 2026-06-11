package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CurrentVersion = 1

type AppConfig struct {
	Version         int         `json:"version"`
	Local           LocalConfig `json:"local"`
	ActiveProfileID string      `json:"active_profile_id"`
	AutoStart       bool        `json:"auto_start"`
	Profiles        []Profile   `json:"profiles"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type LocalConfig struct {
	APIAddress      string `json:"api_address"`
	SOCKSAddress    string `json:"socks_address"`
	AllowLAN        bool   `json:"allow_lan"`
	Mode            string `json:"mode"`
	SystemProxyMode string `json:"system_proxy_mode,omitempty"`
}

type Profile struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Group            string            `json:"group,omitempty"`
	Transport        string            `json:"transport"`
	Endpoint         string            `json:"endpoint"`
	ClientID         string            `json:"client_id"`
	ClientSecret     string            `json:"client_secret"`
	ServerPublicKey  string            `json:"server_public_key"`
	ServerName       string            `json:"server_name,omitempty"`
	SkipTLSVerify    bool              `json:"skip_tls_verify,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	HandshakePadding PaddingConfig     `json:"handshake_padding,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	Remark           string            `json:"remark,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type PaddingConfig struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func Default() AppConfig {
	now := time.Now().UTC()
	return AppConfig{
		Version: CurrentVersion,
		Local: LocalConfig{
			APIAddress:      "127.0.0.1:9727",
			SOCKSAddress:    "127.0.0.1:7890",
			Mode:            "rule",
			SystemProxyMode: "unchanged",
		},
		AutoStart: false,
		Profiles:  []Profile{},
		UpdatedAt: now,
	}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "krayN", "config.json"), nil
}

func Load(path string) (AppConfig, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return AppConfig{}, err
		}
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		return cfg, nil
	}
	if err != nil {
		return AppConfig{}, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return AppConfig{}, err
	}
	Normalize(&cfg)
	return cfg, nil
}

func Save(path string, cfg AppConfig) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	Normalize(&cfg)
	cfg.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func Normalize(cfg *AppConfig) {
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	if cfg.Local.APIAddress == "" {
		cfg.Local.APIAddress = "127.0.0.1:9727"
	}
	if cfg.Local.SOCKSAddress == "" {
		cfg.Local.SOCKSAddress = "127.0.0.1:7890"
	}
	if cfg.Local.Mode == "" {
		cfg.Local.Mode = "rule"
	}
	cfg.Local.Mode = NormalizeMode(cfg.Local.Mode)
	cfg.Local.SystemProxyMode = NormalizeSystemProxyMode(cfg.Local.SystemProxyMode)
	for i := range cfg.Profiles {
		if cfg.Profiles[i].ID == "" {
			cfg.Profiles[i].ID = NewID()
		}
		cfg.Profiles[i].Transport = NormalizeTransport(cfg.Profiles[i].Transport)
		if cfg.Profiles[i].UpdatedAt.IsZero() {
			cfg.Profiles[i].UpdatedAt = time.Now().UTC()
		}
	}
	if cfg.ActiveProfileID == "" && len(cfg.Profiles) > 0 {
		cfg.ActiveProfileID = cfg.Profiles[0].ID
	}
}

func NormalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "global", "direct", "rule":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "rule"
	}
}

func NormalizeSystemProxyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "auto", "pac", "unchanged", "clear":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "unchanged"
	}
}

func Validate(cfg AppConfig) error {
	if err := validateListenAddress(cfg.Local.APIAddress); err != nil {
		return fmt.Errorf("api address: %w", err)
	}
	if err := validateListenAddress(cfg.Local.SOCKSAddress); err != nil {
		return fmt.Errorf("socks address: %w", err)
	}
	ids := make(map[string]struct{}, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		if err := ValidateProfile(profile); err != nil {
			return fmt.Errorf("profile %q: %w", profile.Name, err)
		}
		if _, ok := ids[profile.ID]; ok {
			return fmt.Errorf("duplicate profile id %q", profile.ID)
		}
		ids[profile.ID] = struct{}{}
	}
	if cfg.ActiveProfileID != "" {
		if _, ok := ids[cfg.ActiveProfileID]; !ok {
			return fmt.Errorf("active profile %q does not exist", cfg.ActiveProfileID)
		}
	}
	return nil
}

func ValidateProfile(profile Profile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return errors.New("name is required")
	}
	if NormalizeTransport(profile.Transport) == "" {
		return errors.New("transport is required")
	}
	if strings.TrimSpace(profile.Endpoint) == "" {
		return errors.New("endpoint is required")
	}
	if strings.TrimSpace(profile.ClientID) == "" {
		return errors.New("client_id is required")
	}
	if strings.TrimSpace(profile.ClientSecret) == "" {
		return errors.New("client_secret is required")
	}
	if strings.TrimSpace(profile.ServerPublicKey) == "" {
		return errors.New("server_public_key is required")
	}
	if profile.HandshakePadding.Min < 0 || profile.HandshakePadding.Max < 0 || profile.HandshakePadding.Min > profile.HandshakePadding.Max {
		return errors.New("invalid handshake padding")
	}
	return nil
}

func NormalizeTransport(transport string) string {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "tcp":
		return "tcp"
	case "tls":
		return "tls"
	case "ws", "websocket":
		return "websocket"
	case "http-upgrade", "httpupgrade", "upgrade":
		return "http-upgrade"
	case "http-stream", "httpstream", "http":
		return "http-stream"
	case "grpc":
		return "grpc"
	case "xhttp":
		return "xhttp"
	default:
		return strings.ToLower(strings.TrimSpace(transport))
	}
}

func NewID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("profile-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func FindProfile(cfg AppConfig, id string) (Profile, bool) {
	for _, profile := range cfg.Profiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return Profile{}, false
}

func UpsertProfile(cfg *AppConfig, profile Profile) Profile {
	if profile.ID == "" {
		profile.ID = NewID()
	}
	profile.Transport = NormalizeTransport(profile.Transport)
	profile.UpdatedAt = time.Now().UTC()
	for i := range cfg.Profiles {
		if cfg.Profiles[i].ID == profile.ID {
			cfg.Profiles[i] = profile
			return profile
		}
	}
	cfg.Profiles = append(cfg.Profiles, profile)
	if cfg.ActiveProfileID == "" {
		cfg.ActiveProfileID = profile.ID
	}
	return profile
}

func DeleteProfile(cfg *AppConfig, id string) bool {
	for i := range cfg.Profiles {
		if cfg.Profiles[i].ID == id {
			cfg.Profiles = append(cfg.Profiles[:i], cfg.Profiles[i+1:]...)
			if cfg.ActiveProfileID == id {
				cfg.ActiveProfileID = ""
				if len(cfg.Profiles) > 0 {
					cfg.ActiveProfileID = cfg.Profiles[0].ID
				}
			}
			return true
		}
	}
	return false
}

func validateListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if port == "" {
		return errors.New("port is required")
	}
	if host == "" {
		return errors.New("host is required")
	}
	return nil
}
