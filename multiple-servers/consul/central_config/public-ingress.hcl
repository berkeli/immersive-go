Kind = "ingress-gateway"
Name = "public-ingress"

Listeners = [
  {
    Port = 8082
    Protocol = "http"
    Services = [
      {
        Name = "static"
      }
    ]
  }
]
