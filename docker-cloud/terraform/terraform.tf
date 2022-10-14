provider "aws" {
  region = "eu-west-2"
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

