resource "aws_iam_role" "GitHubActionECRPublicPushImage" {

  name = "GitHubActionECRPublicPushImage-${var.user}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action : [
          "sts:AssumeRoleWithWebIdentity"
        ],
        Effect = "Allow"
        Principal = {
          Federated = data.aws_iam_openid_connect_provider.default.arn,

        }
        Condition = {
          StringEquals = {
            "${data.aws_iam_openid_connect_provider.default.url}:aud" = "sts.amazonaws.com"
          },
          StringLike = {
            "${data.aws_iam_openid_connect_provider.default.url}:sub" = "repo:berkeli/immersive-go:*"
          }
        }
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "GetAuthorizationToken" {
  role       = aws_iam_role.GitHubActionECRPublicPushImage.name
  policy_arn = aws_iam_policy.GetAuthorizationToken.arn
}

resource "aws_iam_role_policy_attachment" "AllowPush" {
  role       = aws_iam_role.GitHubActionECRPublicPushImage.name
  policy_arn = aws_iam_policy.AllowPush.arn
}
