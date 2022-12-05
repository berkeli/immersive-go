Kind = "service-router"
Name = "static"

Routes = [
  {
    Match {
      HTTP {
        PathPrefix = "/api"
      }
    }

    Destination {
      Service = "api"
    }
  }
]
