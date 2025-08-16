# Generated from sample-terraform.tfstate file
# This is a simplified representation for diagram generation

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "main-vpc"
    Environment = "production"
    Project     = "driftmgr-demo"
  }
}

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "us-east-1a"
  map_public_ip_on_launch = true

  tags = {
    Name        = "public-subnet"
    Environment = "production"
    Type        = "public"
  }
}

resource "aws_security_group" "web" {
  name        = "web-security-group"
  description = "Security group for web servers"
  vpc_id      = aws_vpc.main.id

  tags = {
    Name     = "web-sg"
    Environment = "production"
    Purpose  = "web-server"
  }
}

resource "aws_instance" "web_server" {
  ami                    = "ami-0c55b159cbfafe1d0"
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.web.id]

  tags = {
    Name        = "web-server-1"
    Environment = "production"
    Role        = "web-server"
    Owner       = "devops-team"
  }
}
