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
