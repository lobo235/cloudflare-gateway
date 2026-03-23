# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.0.0] - 2026-03-23

### Added

- Initial implementation of cloudflare-gateway HTTP API
- Config loading from environment variables (CF_API_TOKEN, CF_ZONE_ID, GATEWAY_API_KEY, PORT, LOG_LEVEL)
- Cloudflare API client wrapper using `github.com/cloudflare/cloudflare-go`
- GET /health — unauthenticated health check with Cloudflare API reachability
- GET /zones — list accessible Cloudflare zones
- GET /zones/{zoneID}/records — list DNS records with optional type and name filters
- POST /zones/{zoneID}/records — create DNS record
- GET /zones/{zoneID}/records/{recordID} — get record by ID
- PUT /zones/{zoneID}/records/{recordID} — update DNS record
- DELETE /zones/{zoneID}/records/{recordID} — delete DNS record
- GET /zones-by-name/{zoneName}/records — convenience: list records by zone name
- POST /zones-by-name/{zoneName}/records — convenience: create record by zone name
- DELETE /zones-by-name/{zoneName}/records/{recordName} — convenience: delete record by name
- Bearer token authentication middleware with constant-time compare
- X-Trace-ID header propagation (generate UUID if absent)
- Structured JSON logging via log/slog
- Access-log style request logging middleware
- Graceful shutdown on SIGINT/SIGTERM (30s drain)
- Dockerfile with multi-stage build (golang:1.24-alpine → alpine:3.21)
- Makefile with build, test, cover, lint, run, hooks, clean targets
- golangci-lint config with strict linter set
- Pre-commit hook running lint + tests
- Full test suite for config, API handlers, and Cloudflare client
