job "cloudflare-gateway" {
  node_pool   = "default"
  datacenters = ["dc1"]
  type        = "service"

  group "cloudflare-gateway" {
    count = 1

    network {
      port "http" {
        to = 8080
      }
    }

    service {
      name     = "cloudflare-gateway"
      port     = "http"
      provider = "consul"
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.cloudflare-gateway.rule=Host(`cloudflare-gateway.example.com`)",
        "traefik.http.routers.cloudflare-gateway.entrypoints=websecure",
        "traefik.http.routers.cloudflare-gateway.tls=true",
      ]

      check {
        type     = "http"
        path     = "/health"
        port     = "http"
        interval = "30s"
        timeout  = "5s"

        check_restart {
          limit = 3
          grace = "30s"
        }
      }
    }

    restart {
      attempts = 3
      interval = "2m"
      delay    = "15s"
      mode     = "fail"
    }

    vault {
      cluster     = "default"
      change_mode = "restart"
    }

    task "cloudflare-gateway" {
      driver = "docker"

      config {
        image = "ghcr.io/lobo235/cloudflare-gateway:latest"
        ports = ["http"]
      }

      template {
        data = <<EOF
{{ with secret "kv/data/nomad/default/cloudflare-gateway" }}
CF_API_TOKEN={{ .Data.data.cf_api_token }}
GATEWAY_API_KEY={{ .Data.data.gateway_api_key }}
{{ end }}
EOF
        destination = "secrets/cloudflare-gateway.env"
        env         = true
      }

      env {
        PORT      = "8080"
        LOG_LEVEL = "info"
      }

      resources {
        cpu    = 100
        memory = 64
      }

      kill_timeout = "35s"
    }
  }
}
