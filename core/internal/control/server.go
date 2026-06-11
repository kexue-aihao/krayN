package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"krayn/core/internal/config"
	"krayn/core/internal/engine"
)

type Server struct {
	Address string
	Engine  *engine.Engine
	Logger  *slog.Logger

	httpServer *http.Server
}

type response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (s *Server) Start(ctx context.Context) error {
	if s.Engine == nil {
		return errors.New("engine is required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /state", s.handleState)
	mux.HandleFunc("GET /proxy.pac", s.handleProxyPAC)
	mux.HandleFunc("POST /start", s.handleStart)
	mux.HandleFunc("POST /stop", s.handleStop)
	mux.HandleFunc("GET /config", s.handleConfig)
	mux.HandleFunc("PUT /local", s.handleUpdateLocal)
	mux.HandleFunc("GET /profiles", s.handleProfiles)
	mux.HandleFunc("POST /profiles", s.handleUpsertProfile)
	mux.HandleFunc("PUT /profiles/{id}", s.handleUpsertProfile)
	mux.HandleFunc("DELETE /profiles/{id}", s.handleDeleteProfile)
	mux.HandleFunc("POST /profiles/{id}/activate", s.handleActivateProfile)

	s.httpServer = &http.Server{
		Addr:              s.Address,
		Handler:           cors(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "krayn-core",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Engine.State())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Engine.Config())
}

func (s *Server) handleProxyPAC(w http.ResponseWriter, r *http.Request) {
	cfg := s.Engine.Config()
	host, port, err := net.SplitHostPort(cfg.Local.SOCKSAddress)
	if err != nil || host == "" || port == "" {
		host = "127.0.0.1"
		port = "7890"
	}
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `function FindProxyForURL(url, host) {
  if (isPlainHostName(host) ||
      shExpMatch(host, "localhost") ||
      shExpMatch(host, "127.*") ||
      shExpMatch(host, "10.*") ||
      shExpMatch(host, "192.168.*") ||
      shExpMatch(host, "172.16.*") ||
      shExpMatch(host, "172.17.*") ||
      shExpMatch(host, "172.18.*") ||
      shExpMatch(host, "172.19.*") ||
      shExpMatch(host, "172.20.*") ||
      shExpMatch(host, "172.21.*") ||
      shExpMatch(host, "172.22.*") ||
      shExpMatch(host, "172.23.*") ||
      shExpMatch(host, "172.24.*") ||
      shExpMatch(host, "172.25.*") ||
      shExpMatch(host, "172.26.*") ||
      shExpMatch(host, "172.27.*") ||
      shExpMatch(host, "172.28.*") ||
      shExpMatch(host, "172.29.*") ||
      shExpMatch(host, "172.30.*") ||
      shExpMatch(host, "172.31.*")) {
    return "DIRECT";
  }
  return "SOCKS5 %s:%s; SOCKS %s:%s; DIRECT";
}
`, host, port, host, port)
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Engine.Config().Profiles)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if err := s.Engine.Start(context.Background()); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.Engine.State())
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if err := s.Engine.Stop(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, s.Engine.State())
}

func (s *Server) handleUpdateLocal(w http.ResponseWriter, r *http.Request) {
	var local config.LocalConfig
	if err := json.NewDecoder(r.Body).Decode(&local); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Engine.UpdateLocal(local); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.Engine.Config().Local)
}

func (s *Server) handleUpsertProfile(w http.ResponseWriter, r *http.Request) {
	var profile config.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id := r.PathValue("id")
	if id != "" {
		profile.ID = id
	}
	saved, err := s.Engine.UpsertProfile(profile)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := s.Engine.DeleteProfile(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, response{OK: true})
}

func (s *Server) handleActivateProfile(w http.ResponseWriter, r *http.Request) {
	if err := s.Engine.SetActiveProfile(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, s.Engine.State())
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, response{OK: false, Error: err.Error()})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
