variable "image_tag" {
  description = "The image tag to use for the docker-cloud container"
  default     = "latest"
}

variable "aws_profile" {
  description = "The AWS profile to use for terraform"
  default     = "personal"
}

