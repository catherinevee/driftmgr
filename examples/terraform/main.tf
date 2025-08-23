terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.medium"
  
  tags = {
    Name        = "WebServer"
    Environment = "production"
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket-12345"
  
  tags = {
    Environment = "production"
    Purpose     = "data-storage"
  }
}