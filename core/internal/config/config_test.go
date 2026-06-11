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

	Normalize(&cfg)

	if cfg.Local.Mode != "global" {
		t.Fatalf("mode mismatch: got %q", cfg.Local.Mode)
	}
	if cfg.Local.SystemProxyMode != "pac" {
		t.Fatalf("system proxy mode mismatch: got %q", cfg.Local.SystemProxyMode)
	}

	cfg.Local.Mode = "unexpected"
	cfg.Local.SystemProxyMode = "unexpected"
	Normalize(&cfg)
	if cfg.Local.Mode != "rule" {
		t.Fatalf("invalid mode should default to rule, got %q", cfg.Local.Mode)
	}
	if cfg.Local.SystemProxyMode != "unchanged" {
		t.Fatalf("invalid system proxy mode should default to unchanged, got %q", cfg.Local.SystemProxyMode)
	}
}
