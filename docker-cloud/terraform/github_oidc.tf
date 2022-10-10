data "tls_certificate" "github_actions_oidc_endpoint" {
  url = "https://token.actions.githubusercontent.com"
}

resource "aws_iam_openid_connect_provider" "default" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = [
    "sts.amazonaws.com",
  ]

  thumbprint_list = [
    data.tls_certificate.github_actions_oidc_endpoint.certificates.0.sha1_fingerprint,
  ]
}
