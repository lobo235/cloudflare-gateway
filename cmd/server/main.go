package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lobo235/cloudflare-gateway/internal/api"
	cfclient "github.com/lobo235/cloudflare-gateway/internal/cloudflare"
	"github.com/lobo235/cloudflare-gateway/internal/config"
)

// version is set at build time via -ldflags "-X main.version=<value>".
var version = "dev"

func main() {
	// Bootstrap logger at INFO until config is loaded.
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("starting cloudflare-gateway", "version", version)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config error", "error", err)
		os.Exit(1)
	}

	// Reconfigure logger with the level from config.
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()}))

	client, err := cfclient.NewClient(cfg.CFAPIToken)
	if err != nil {
		log.Error("failed to create cloudflare client", "error", err)
		os.Exit(1)
	}

	if cfg.CFZoneID != "" {
		log.Info("configured default zone", "zone_id", cfg.CFZoneID)
	}

	srv := api.NewServer(client, cfg.GatewayAPIKey, version, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := ":" + cfg.Port
	if err := srv.Run(ctx, addr); err != nil {
		log.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
