service {
  name    = "server"
  id      = "server_2"
  address = "10.20.0.4"
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
    }
  }
}