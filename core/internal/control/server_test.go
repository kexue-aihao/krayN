package control

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"krayn/core/internal/config"
	"krayn/core/internal/engine"
)

func TestProxyPACUsesConfiguredSOCKSAddress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := config.Default()
	cfg.Local.SOCKSAddress = "127.0.0.1:7899"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	appEngine, err := engine.New(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Engine: appEngine}
	req := httptest.NewRequest(http.MethodGet, "/proxy.pac", nil)
	rec := httptest.NewRecorder()

	server.handleProxyPAC(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SOCKS5 127.0.0.1:7899") {
		t.Fatalf("PAC body does not contain configured SOCKS address: %s", body)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/x-ns-proxy-autoconfig") {
		t.Fatalf("content type mismatch: %s", rec.Header().Get("Content-Type"))
	}
}

func TestProfileDiagnosticsRejectsMalformedBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := config.Default()
	profile := config.UpsertProfile(&cfg, config.Profile{
		Name:            "local relay",
		Transport:       "tcp",
		Endpoint:        "127.0.0.1:9000",
		ClientID:        "demo",
		ClientSecret:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ServerPublicKey: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	cfg.ActiveProfileID = profile.ID
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	appEngine, err := engine.New(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Engine: appEngine}
	req := httptest.NewRequest(http.MethodPost, "/profiles/"+profile.ID+"/diagnostics", strings.NewReader("{bad"))
	req.SetPathValue("id", profile.ID)
	rec := httptest.NewRecorder()

	server.handleProfileDiagnostics(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d", rec.Code)
	}
}
