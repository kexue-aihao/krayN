package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kray/pkg/kless"
	"krayn/core/internal/config"
	"krayn/core/internal/control"
	"krayn/core/internal/engine"
)

var (
	version = "dev"
	commit  = "local"
)

func main() {
	os.Exit(run())
}

func run() int {
	var configPath string
	var printVersion bool
	var generateKeys bool
	flag.StringVar(&configPath, "config", "", "path to krayN config.json")
	flag.BoolVar(&printVersion, "version", false, "print version information")
	flag.BoolVar(&generateKeys, "gen-keys", false, "generate KLESS demo identity material")
	flag.Parse()

	if printVersion {
		fmt.Printf("krayn-core %s (%s)\n", version, commit)
		return 0
	}
	if generateKeys {
		if err := printKLESSKeys(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	appEngine, err := engine.New(configPath, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := appEngine.Config()
	if cfg.AutoStart && cfg.ActiveProfileID != "" {
		if err := appEngine.Start(ctx); err != nil {
			logger.Warn("autostart failed", "error", err)
		}
	}

	server := &control.Server{
		Address: cfg.Local.APIAddress,
		Engine:  appEngine,
		Logger:  logger,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("control api listening", "address", cfg.Local.APIAddress)
		errCh <- server.Start(ctx)
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			logger.Error("control api stopped", "error", err)
			return 1
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = appEngine.Stop()
	_ = server.Shutdown(shutdownCtx)
	return 0
}

func printKLESSKeys() error {
	serverPublic, serverPrivate, err := kless.GenerateServerIdentity()
	if err != nil {
		return err
	}
	clientSecret, err := kless.GenerateClientSecret()
	if err != nil {
		return err
	}
	fmt.Printf("server_public_key=%s\n", kless.EncodeKey(serverPublic))
	fmt.Printf("server_private_key=%s\n", kless.EncodeKey(serverPrivate))
	fmt.Printf("client_secret=%s\n", kless.EncodeKey(clientSecret))
	fmt.Printf("client_id=krayn-%d\n", time.Now().Unix())
	return nil
}

func init() {
	if _, err := config.DefaultPath(); err != nil {
		return
	}
}
