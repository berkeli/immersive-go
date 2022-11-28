service {
    name = "psql"
    id = "psql"
    ip = "10.5.0.3"
    port = 5432
    tags = ["primary", "v1"]

    connect {
        sidecar_service {
            port = 20000

            check {
                name     = "Connect Envoy Sidecar"
                tcp      = "10.5.0.3:20000"
                interval = "10s"
                timeout  = "2s"
            }
        }
    }
}