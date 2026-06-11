package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := Default()
	profile := UpsertProfile(&cfg, Profile{
		Name:            "local relay",
		Transport:       "tcp",
		Endpoint:        "127.0.0.1:9000",
		ClientID:        "demo",
		ClientSecret:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ServerPublicKey: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	cfg.ActiveProfileID = profile.ID
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveProfileID != profile.ID {
		t.Fatalf("active profile mismatch: got %q want %q", loaded.ActiveProfileID, profile.ID)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("profile count mismatch: got %d", len(loaded.Profiles))
	}
}

func TestNormalizeLocalModes(t *testing.T) {
	cfg := Default()
	cfg.Local.Mode = "GLOBAL"
	cfg.Local.SystemProxyMode = "PAC"
	cfg.Local.ResolverType = "HTTPS"
	cfg.Local.ResolverAddress = " https://doh.pub/dns-query "

	Normalize(&cfg)

	if cfg.Local.Mode != "global" {
		t.Fatalf("mode mismatch: got %q", cfg.Local.Mode)
	}
	if cfg.Local.SystemProxyMode != "pac" {
		t.Fatalf("system proxy mode mismatch: got %q", cfg.Local.SystemProxyMode)
	}
	if cfg.Local.ResolverType != "doh" {
		t.Fatalf("resolver type mismatch: got %q", cfg.Local.ResolverType)
	}
	if cfg.Local.ResolverAddress != "https://doh.pub/dns-query" {
		t.Fatalf("resolver address mismatch: got %q", cfg.Local.ResolverAddress)
	}

	cfg.Local.Mode = "unexpected"
	cfg.Local.SystemProxyMode = "unexpected"
	cfg.Local.ResolverType = "unexpected"
	Normalize(&cfg)
	if cfg.Local.Mode != "rule" {
		t.Fatalf("invalid mode should default to rule, got %q", cfg.Local.Mode)
	}
	if cfg.Local.SystemProxyMode != "unchanged" {
		t.Fatalf("invalid system proxy mode should default to unchanged, got %q", cfg.Local.SystemProxyMode)
	}
	if cfg.Local.ResolverType != "system" {
		t.Fatalf("invalid resolver type should default to system, got %q", cfg.Local.ResolverType)
	}
}

func TestValidateResolver(t *testing.T) {
	cfg := Default()
	cfg.Local.ResolverType = "dns"
	cfg.Local.ResolverAddress = "119.29.29.29"
	if err := Validate(cfg); err != nil {
		t.Fatalf("dns resolver should validate: %v", err)
	}

	cfg.Local.ResolverType = "doh"
	cfg.Local.ResolverAddress = "http://example.com/dns-query"
	if err := Validate(cfg); err == nil {
		t.Fatal("insecure doh url should be rejected")
	}

	cfg.Local.ResolverType = "dns"
	cfg.Local.ResolverAddress = "2400:3200::1"
	if err := Validate(cfg); err != nil {
		t.Fatalf("bare IPv6 dns resolver should validate: %v", err)
	}
}
