package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/lobo235/cloudflare-gateway/internal/cloudflare"
)

// cloudflareClient is the interface the Server uses to communicate with Cloudflare.
// The concrete *cloudflare.Client satisfies this interface.
type cloudflareClient interface {
	Ping(ctx context.Context) error
	ListZones(ctx context.Context) ([]cloudflare.Zone, error)
	GetZoneIDByName(ctx context.Context, zoneName string) (string, error)
	ListDNSRecords(ctx context.Context, zoneID, recordType, recordName string) ([]cloudflare.DNSRecord, error)
	GetDNSRecord(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error)
	CreateDNSRecord(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error)
	UpdateDNSRecord(ctx context.Context, zoneID, recordID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error)
	DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error
}

// Server holds the dependencies for the HTTP server.
type Server struct {
	cf      cloudflareClient
	apiKey  string
	version string
	log     *slog.Logger
}

// NewServer creates a Server wired to the given Cloudflare client, API key, version string, and logger.
func NewServer(client cloudflareClient, apiKey, version string, log *slog.Logger) *Server {
	return &Server{
		cf:      client,
		apiKey:  apiKey,
		version: version,
		log:     log,
	}
}

// Handler builds and returns the root http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	auth := bearerAuth(s.apiKey)

	// /health is unauthenticated — used by Nomad container health checks
	mux.HandleFunc("GET /health", s.healthHandler())

	// Zone routes
	mux.Handle("GET /zones", auth(http.HandlerFunc(s.listZonesHandler())))

	// DNS record routes (by zone ID)
	mux.Handle("GET /zones/{zoneID}/records", auth(http.HandlerFunc(s.listRecordsHandler())))
	mux.Handle("POST /zones/{zoneID}/records", auth(http.HandlerFunc(s.createRecordHandler())))
	mux.Handle("GET /zones/{zoneID}/records/{recordID}", auth(http.HandlerFunc(s.getRecordHandler())))
	mux.Handle("PUT /zones/{zoneID}/records/{recordID}", auth(http.HandlerFunc(s.updateRecordHandler())))
	mux.Handle("DELETE /zones/{zoneID}/records/{recordID}", auth(http.HandlerFunc(s.deleteRecordHandler())))

	// Convenience routes (by zone name) — separate top-level to avoid mux conflicts
	mux.Handle("GET /zones-by-name/{zoneName}/records", auth(http.HandlerFunc(s.listRecordsByZoneNameHandler())))
	mux.Handle("POST /zones-by-name/{zoneName}/records", auth(http.HandlerFunc(s.createRecordByZoneNameHandler())))
	mux.Handle("DELETE /zones-by-name/{zoneName}/records/{recordName}", auth(http.HandlerFunc(s.deleteRecordByZoneNameHandler())))

	return requestLogger(s.log)(mux)
}

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
