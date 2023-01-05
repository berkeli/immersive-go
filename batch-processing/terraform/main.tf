terraform {

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "eu-west-2"

  profile = "cyfplus"

  default_tags {
    tags = {
      project   = "batch-processing"
      owner     = "berkeli"
      terraform = "true"
    }
  }
}
