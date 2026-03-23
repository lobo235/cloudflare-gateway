job "cloudflare-gateway" {
  datacenters = ["dc1"]
  type        = "service"

  group "cloudflare-gateway" {
    count = 1

    network {
      port "http" {
        static = 8080
      }
    }

    task "cloudflare-gateway" {
      driver = "docker"

      config {
        image = "ghcr.io/lobo235/cloudflare-gateway:latest"
        ports = ["http"]
      }

      env {
        PORT = "${NOMAD_PORT_http}"
      }

      template {
        data        = <<-EOF
          {{ with nomadVar "nomad/jobs/cloudflare-gateway" }}
          CF_API_TOKEN={{ .cf_api_token }}
          GATEWAY_API_KEY={{ .gateway_api_key }}
          CF_ZONE_ID={{ .cf_zone_id }}
          LOG_LEVEL={{ .log_level }}
          {{ end }}
        EOF
        destination = "secrets/env.env"
        env         = true
      }

      resources {
        cpu    = 100
        memory = 64
      }

      service {
        name = "cloudflare-gateway"
        port = "http"

        check {
          type     = "http"
          path     = "/health"
          interval = "15s"
          timeout  = "5s"
        }
      }
    }
  }
}
