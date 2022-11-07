resource "aws_s3_bucket" "b" {
  bucket = "batch-processing-berkeli"
}

resource "aws_s3_bucket_policy" "public_access" {
  bucket = aws_s3_bucket.b.id
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicAccess",
      "Effect": "Allow",
      "Principal": "*",
      "Action": ["s3:GetObject", "s3:GetObjectVersion"],
      "Resource": "arn:aws:s3:::${aws_s3_bucket.b.id}/*"
    }
  ]
}
POLICY
}
