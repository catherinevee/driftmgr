terraform {
  required_version = ">= 1.0"
  
  cloud {
    organization = "your-organization"
    
    workspaces {
      name = "driftmgr-infrastructure"
    }
  }
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
    
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

# Provider configurations
provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Environment = var.environment
      ManagedBy   = "terraform"
      Project     = "driftmgr"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
}

# Variables
variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "gcp_project_id" {
  description = "GCP project ID"
  type        = string
  default     = "your-gcp-project"
}

variable "gcp_region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

# DriftMgr configuration
resource "null_resource" "driftmgr_config" {
  triggers = {
    environment = var.environment
    timestamp   = timestamp()
  }
  
  provisioner "local-exec" {
    command = <<-EOT
      cat > driftmgr.yaml << EOF
      providers:
        aws:
          regions: [${var.aws_region}]
        azure:
          regions: [eastus, westus]
        gcp:
          regions: [${var.gcp_region}]
      ci_cd:
        enabled: true
        fail_on_drift: true
        auto_remediate: false
        severity_threshold: high
        environments:
          ${var.environment}:
            fail_on_drift: true
            auto_remediate: false
      EOF
    EOT
  }
}

# Drift detection hook
resource "null_resource" "drift_detection" {
  depends_on = [null_resource.driftmgr_config]
  
  triggers = {
    # Trigger on any infrastructure change
    infrastructure_hash = sha1(jsonencode([
      for resource in data.terraform_remote_state.infrastructure.outputs : resource
    ]))
  }
  
  provisioner "local-exec" {
    command = <<-EOT
      echo "Running drift detection..."
      
      # Install DriftMgr if not present
      if ! command -v driftmgr &> /dev/null; then
        echo "Installing DriftMgr..."
        git clone https://github.com/catherinevee/driftmgr.git
        cd driftmgr
        make build
        sudo cp bin/driftmgr /usr/local/bin/
        cd ..
      fi
      
      # Discover resources
      driftmgr discover aws ${var.aws_region}
      
      # Analyze drift
      driftmgr analyze terraform.tfstate --output json > drift-analysis.json
      
      # Parse results
      DRIFT_COUNT=$(jq '.drift_count // 0' drift-analysis.json)
      
      if [ "$DRIFT_COUNT" -gt 0 ]; then
        echo "❌ Drift detected: $DRIFT_COUNT resources"
        echo "Drift details:"
        jq '.drifts[] | "  - \(.resource_name) (\(.resource_type)): \(.description)"' drift-analysis.json
        
        # Send notification
        driftmgr notify slack "Terraform Cloud Drift Alert" "Drift detected in ${var.environment} environment"
        
        # Fail the apply if drift is detected
        exit 1
      else
        echo "✅ No drift detected"
      fi
    EOT
    
    environment = {
      AWS_ACCESS_KEY_ID     = var.aws_access_key_id
      AWS_SECRET_ACCESS_KEY = var.aws_secret_access_key
      SLACK_WEBHOOK_URL     = var.slack_webhook_url
    }
  }
}

# Post-apply verification
resource "null_resource" "post_apply_verification" {
  depends_on = [null_resource.drift_detection]
  
  triggers = {
    # Always run after apply
    apply_timestamp = timestamp()
  }
  
  provisioner "local-exec" {
    command = <<-EOT
      echo "Verifying deployment..."
      
      # Wait for resources to be fully provisioned
      sleep 30
      
      # Discover resources after deployment
      driftmgr discover aws ${var.aws_region}
      
      # Check for post-deployment drift
      driftmgr analyze terraform.tfstate --output json > post-deployment-drift.json
      
      POST_DRIFT_COUNT=$(jq '.drift_count // 0' post-deployment-drift.json)
      
      if [ "$POST_DRIFT_COUNT" -gt 0 ]; then
        echo "⚠️ Post-deployment drift detected: $POST_DRIFT_COUNT resources"
        driftmgr notify slack "Post-Deployment Drift" "Drift detected after Terraform Cloud deployment"
      else
        echo "✅ Deployment verified successfully"
      fi
    EOT
    
    environment = {
      AWS_ACCESS_KEY_ID     = var.aws_access_key_id
      AWS_SECRET_ACCESS_KEY = var.aws_secret_access_key
      SLACK_WEBHOOK_URL     = var.slack_webhook_url
    }
  }
}

# Variables for credentials (set in Terraform Cloud)
variable "aws_access_key_id" {
  description = "AWS access key ID"
  type        = string
  sensitive   = true
}

variable "aws_secret_access_key" {
  description = "AWS secret access key"
  type        = string
  sensitive   = true
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for notifications"
  type        = string
  sensitive   = true
  default     = ""
}

# Data source for existing infrastructure state
data "terraform_remote_state" "infrastructure" {
  backend = "remote"
  
  config = {
    organization = "your-organization"
    workspaces = {
      name = "infrastructure"
    }
  }
}

# Outputs
output "drift_detection_status" {
  description = "Status of drift detection"
  value       = "Drift detection completed successfully"
}

output "environment" {
  description = "Environment name"
  value       = var.environment
}

# Example infrastructure resources
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  
  tags = {
    Name = "${var.environment}-vpc"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
  
  tags = {
    Name = "${var.environment}-subnet"
  }
}

# Azure example resource
resource "azurerm_resource_group" "main" {
  name     = "${var.environment}-rg"
  location = "East US"
}

# GCP example resource
resource "google_compute_network" "main" {
  name                    = "${var.environment}-network"
  auto_create_subnetworks = false
}

# DriftMgr monitoring configuration
resource "null_resource" "driftmgr_monitoring" {
  depends_on = [null_resource.post_apply_verification]
  
  triggers = {
    # Run every hour
    hourly = formatdate("YYYY-MM-DD-HH", timestamp())
  }
  
  provisioner "local-exec" {
    command = <<-EOT
      echo "Running scheduled drift monitoring..."
      
      # Create monitoring directory
      mkdir -p drift-monitoring
      
      # Run drift check
      driftmgr discover aws ${var.aws_region}
      driftmgr analyze terraform.tfstate --output json > drift-monitoring/scheduled-check.json
      
      DRIFT_COUNT=$(jq '.drift_count // 0' drift-monitoring/scheduled-check.json)
      
      if [ "$DRIFT_COUNT" -gt 0 ]; then
        echo "🚨 Scheduled drift check: $DRIFT_COUNT resources with drift"
        driftmgr notify slack "Scheduled Drift Alert" "Drift detected in ${var.environment} environment"
        
        # Generate detailed report
        driftmgr export terraform html --output drift-monitoring/drift-report.html
        driftmgr export terraform json --output drift-monitoring/drift-report.json
        
        # Optional: Auto-remediate if configured
        if [ "$AUTO_REMEDIATE" = "true" ]; then
          echo "Auto-remediating drift..."
          driftmgr remediate-batch terraform --auto
        fi
      else
        echo "✅ Scheduled drift check: No drift detected"
      fi
    EOT
    
    environment = {
      AWS_ACCESS_KEY_ID     = var.aws_access_key_id
      AWS_SECRET_ACCESS_KEY = var.aws_secret_access_key
      SLACK_WEBHOOK_URL     = var.slack_webhook_url
      AUTO_REMEDIATE        = "false"
    }
  }
}
