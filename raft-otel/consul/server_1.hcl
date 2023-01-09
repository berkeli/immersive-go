service {
  name    = "server"
  id      = "server_1"
  address = "10.20.0.3"
  port    = 50051

  tags = ["v1"]
  meta = {
    version = "1"
  }

  connect {
    sidecar_service {
      port = 20000

      check {
        name     = "Connect Envoy Sidecar"
        tcp      = "10.20.0.3:20000"
        interval = "10s"
      }

      proxy {
        upstreams {
          destination_name   = "server"
          local_bind_address = "127.0.0.1"
          local_bind_port    = 50052

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