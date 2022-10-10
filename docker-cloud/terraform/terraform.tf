provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Environment = "dev"
      Projet      = "docker-cloud"
      terraform   = "true"
    }
  }
}

terraform {
  cloud {
    organization = "berkeli"

    workspaces {
      name = "docker-cloud"
    }
  }
}

