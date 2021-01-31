provider "aws" {
  version = "~> 2.0"

  profile = var.profile
  region  = var.region
}

terraform {
  # The configuration for this backend will be filled in by Terragrunt
  backend "s3" {
  }
}

resource "aws_vpc" "test1" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "foo"
    awsrm = "test-acc"
  }
}

resource "aws_vpc" "test2" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "bar"
    awsrm = "test-acc"
  }
}

resource "aws_vpc" "test3" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "baz"
    awsrm = "test-acc"
  }
}