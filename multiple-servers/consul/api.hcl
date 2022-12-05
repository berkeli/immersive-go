service {
  name    = "api"
  id      = "api"
  address = "10.10.0.5"
  port    = 8081

  tags = ["v1"]
  meta = {
    version = "1"
  }

  connect {
    sidecar_service {
      port = 20000

      check {
        name     = "Connect Envoy Sidecar"
        tcp      = "10.10.0.5:20000"
        interval = "10s"
      }

      proxy {
        upstreams {
          destination_name   = "psql"
          local_bind_address = "127.0.0.1"
          local_bind_port    = 5432

          config {
            protocol = "tcp"
            connect_timeout_ms = 5000
            envoy_prometheus_bind_addr = "10.10.0.5:9102"
            limits {
              max_connections         = 3
              max_pending_requests    = 4
              max_concurrent_requests = 5
            }
            passive_health_check {
              interval     = "30s"
              max_failures = 10
            }
          }
        }
      }
    }
  }
}
