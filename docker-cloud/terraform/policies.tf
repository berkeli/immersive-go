resource "aws_iam_policy" "GetAuthorizationToken" {
  name        = "GetAuthorizationToken-${var.user}"
  path        = "/"
  description = "To get authorization token for github"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action : [
          "ecr-public:GetAuthorizationToken",
          "sts:GetServiceBearerToken"
        ],
        Effect   = "Allow"
        Resource = "*"
      },
    ]
  })
}


resource "aws_iam_policy" "AllowPush" {
  name        = "AllowPush-${var.user}"
  path        = "/"
  description = "To push images to ecr public"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action : [
          "ecr-public:InitiateLayerUpload",
          "ecr-public:UploadLayerPart",
          "ecr-public:PutImage",
          "ecr-public:CompleteLayerUpload",
          "ecr-public:BatchCheckLayerAvailability"
        ],
        Effect   = "Allow"
        Resource = "*"
      },
    ]
  })
}

