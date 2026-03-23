# cloudflare-gateway

Authenticated HTTP API wrapper for Cloudflare DNS record management.
Part of the [homelab-ai](https://github.com/lobo235/homelab-ai) platform.

## Quick Start

```bash
cp .env.example .env
# Fill in CF_API_TOKEN and GATEWAY_API_KEY
go run ./cmd/server
```

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Health check + Cloudflare API reachability |
| GET | `/zones` | Bearer | List accessible zones |
| GET | `/zones/{zoneID}/records` | Bearer | List DNS records (`?type=CNAME&name=foo`) |
| POST | `/zones/{zoneID}/records` | Bearer | Create DNS record |
| GET | `/zones/{zoneID}/records/{recordID}` | Bearer | Get record by ID |
| PUT | `/zones/{zoneID}/records/{recordID}` | Bearer | Update DNS record |
| DELETE | `/zones/{zoneID}/records/{recordID}` | Bearer | Delete DNS record |
| GET | `/zones-by-name/{zoneName}/records` | Bearer | List records by zone name |
| POST | `/zones-by-name/{zoneName}/records` | Bearer | Create record by zone name |
| DELETE | `/zones-by-name/{zoneName}/records/{recordName}` | Bearer | Delete record by name |

## License

Private — part of the homelab-ai platform.
