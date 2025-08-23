# Terraform Cloud Integration for DriftMgr
# This configuration sets up Terraform Cloud workspaces and resources
# for DriftMgr CI/CD pipeline integration

terraform {
  required_version = ">= 1.0"
  
  required_providers {
    tfe = {
      source  = "hashicorp/tfe"
      version = "~> 0.52.0"
    }
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
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
      version = "~> 5.0"
    }
  }

  cloud {
    organization = var.tfc_organization
    
    workspaces {
      name = "driftmgr-infrastructure"
    }
  }
}

# Variables
variable "tfc_organization" {
  description = "Terraform Cloud organization name"
  type        = string
}

variable "github_token" {
  description = "GitHub personal access token"
  type        = string
  sensitive   = true
}

variable "github_repo" {
  description = "GitHub repository (org/repo)"
  type        = string
  default     = "your-org/driftmgr"
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for notifications"
  type        = string
  sensitive   = true
  default     = ""
}

variable "environments" {
  description = "List of environments to manage"
  type        = list(string)
  default     = ["development", "staging", "production"]
}

variable "aws_regions" {
  description = "AWS regions for multi-region deployment"
  type        = list(string)
  default     = ["us-east-1", "us-west-2", "eu-west-1"]
}

variable "azure_regions" {
  description = "Azure regions for multi-region deployment"
  type        = list(string)
  default     = ["East US", "West US 2", "West Europe"]
}

variable "gcp_regions" {
  description = "GCP regions for multi-region deployment"
  type        = list(string)
  default     = ["us-central1", "us-east1", "europe-west1"]
}

# Configure providers
provider "tfe" {
  # Token configured via TFE_TOKEN environment variable
}

provider "github" {
  token = var.github_token
}

# Data sources
data "tfe_organization" "main" {
  name = var.tfc_organization
}

data "github_repository" "driftmgr" {
  full_name = var.github_repo
}

# Terraform Cloud OAuth Client for GitHub integration
resource "tfe_oauth_client" "github" {
  organization     = data.tfe_organization.main.name
  api_url          = "https://api.github.com"
  http_url         = "https://github.com"
  oauth_token      = var.github_token
  service_provider = "github"
}

# Variable Sets for shared configuration
resource "tfe_variable_set" "driftmgr_common" {
  name         = "driftmgr-common"
  description  = "Common variables for DriftMgr workspaces"
  organization = data.tfe_organization.main.name
}

resource "tfe_variable" "slack_webhook" {
  key             = "SLACK_WEBHOOK_URL"
  value           = var.slack_webhook_url
  category        = "env"
  description     = "Slack webhook URL for notifications"
  sensitive       = true
  variable_set_id = tfe_variable_set.driftmgr_common.id
}

resource "tfe_variable" "github_token_var" {
  key             = "GITHUB_TOKEN"
  value           = var.github_token
  category        = "env"
  description     = "GitHub token for API access"
  sensitive       = true
  variable_set_id = tfe_variable_set.driftmgr_common.id
}

# AWS Variable Set
resource "tfe_variable_set" "aws_credentials" {
  name         = "aws-credentials"
  description  = "AWS credentials for DriftMgr"
  organization = data.tfe_organization.main.name
}

resource "tfe_variable" "aws_access_key_id" {
  key             = "AWS_ACCESS_KEY_ID"
  value           = "" # Set externally
  category        = "env"
  description     = "AWS Access Key ID"
  sensitive       = true
  variable_set_id = tfe_variable_set.aws_credentials.id
}

resource "tfe_variable" "aws_secret_access_key" {
  key             = "AWS_SECRET_ACCESS_KEY"
  value           = "" # Set externally
  category        = "env"
  description     = "AWS Secret Access Key"
  sensitive       = true
  variable_set_id = tfe_variable_set.aws_credentials.id
}

# Azure Variable Set
resource "tfe_variable_set" "azure_credentials" {
  name         = "azure-credentials"
  description  = "Azure credentials for DriftMgr"
  organization = data.tfe_organization.main.name
}

resource "tfe_variable" "azure_client_id" {
  key             = "ARM_CLIENT_ID"
  value           = "" # Set externally
  category        = "env"
  description     = "Azure Client ID"
  sensitive       = true
  variable_set_id = tfe_variable_set.azure_credentials.id
}

resource "tfe_variable" "azure_client_secret" {
  key             = "ARM_CLIENT_SECRET"
  value           = "" # Set externally
  category        = "env"
  description     = "Azure Client Secret"
  sensitive       = true
  variable_set_id = tfe_variable_set.azure_credentials.id
}

resource "tfe_variable" "azure_tenant_id" {
  key             = "ARM_TENANT_ID"
  value           = "" # Set externally
  category        = "env"
  description     = "Azure Tenant ID"
  sensitive       = true
  variable_set_id = tfe_variable_set.azure_credentials.id
}

resource "tfe_variable" "azure_subscription_id" {
  key             = "ARM_SUBSCRIPTION_ID"
  value           = "" # Set externally
  category        = "env"
  description     = "Azure Subscription ID"
  sensitive       = true
  variable_set_id = tfe_variable_set.azure_credentials.id
}

# GCP Variable Set
resource "tfe_variable_set" "gcp_credentials" {
  name         = "gcp-credentials"
  description  = "GCP credentials for DriftMgr"
  organization = data.tfe_organization.main.name
}

resource "tfe_variable" "gcp_credentials_json" {
  key             = "GOOGLE_CREDENTIALS"
  value           = "" # Set externally
  category        = "env"
  description     = "GCP Service Account JSON"
  sensitive       = true
  variable_set_id = tfe_variable_set.gcp_credentials.id
}

# DigitalOcean Variable Set
resource "tfe_variable_set" "digitalocean_credentials" {
  name         = "digitalocean-credentials"
  description  = "DigitalOcean credentials for DriftMgr"
  organization = data.tfe_organization.main.name
}

resource "tfe_variable" "do_token" {
  key             = "DIGITALOCEAN_TOKEN"
  value           = "" # Set externally
  category        = "env"
  description     = "DigitalOcean API Token"
  sensitive       = true
  variable_set_id = tfe_variable_set.digitalocean_credentials.id
}

# DriftMgr Infrastructure Workspace
resource "tfe_workspace" "driftmgr_infrastructure" {
  name              = "driftmgr-infrastructure"
  organization      = data.tfe_organization.main.name
  description       = "Main infrastructure workspace for DriftMgr"
  working_directory = "infrastructure"
  terraform_version = "1.6.0"
  
  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.github.oauth_token_id
    branch         = "main"
  }

  trigger_prefixes = [
    "infrastructure/",
    "modules/"
  ]

  queue_all_runs = false
  auto_apply     = false

  tags = ["driftmgr", "infrastructure", "main"]
}

# Environment-specific workspaces
resource "tfe_workspace" "environment_workspaces" {
  for_each = toset(var.environments)

  name              = "driftmgr-${each.value}"
  organization      = data.tfe_organization.main.name
  description       = "DriftMgr ${each.value} environment workspace"
  working_directory = "environments/${each.value}"
  terraform_version = "1.6.0"
  
  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.github.oauth_token_id
    branch         = each.value == "production" ? "main" : "develop"
  }

  trigger_prefixes = [
    "environments/${each.value}/",
    "modules/"
  ]

  queue_all_runs = each.value != "production"
  auto_apply     = each.value == "development"

  tags = ["driftmgr", "environment", each.value]
}

# Drift Detection Workspace
resource "tfe_workspace" "drift_detection" {
  name              = "driftmgr-drift-detection"
  organization      = data.tfe_organization.main.name
  description       = "Workspace for running drift detection scans"
  working_directory = "drift-detection"
  terraform_version = "1.6.0"
  
  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.github.oauth_token_id
    branch         = "main"
  }

  trigger_prefixes = [
    "drift-detection/"
  ]

  queue_all_runs = false
  auto_apply     = false

  tags = ["driftmgr", "drift-detection", "monitoring"]
}

# Regional AWS workspaces for multi-region drift detection
resource "tfe_workspace" "aws_regional" {
  for_each = toset(var.aws_regions)

  name              = "driftmgr-aws-${replace(each.value, "-", "")}"
  organization      = data.tfe_organization.main.name
  description       = "DriftMgr AWS ${each.value} region workspace"
  working_directory = "regions/aws/${each.value}"
  terraform_version = "1.6.0"
  
  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.github.oauth_token_id
    branch         = "main"
  }

  trigger_prefixes = [
    "regions/aws/${each.value}/"
  ]

  queue_all_runs = false
  auto_apply     = false

  tags = ["driftmgr", "aws", "region", each.value]
}

# Associate variable sets with workspaces
resource "tfe_workspace_variable_set" "common_to_infrastructure" {
  workspace_id    = tfe_workspace.driftmgr_infrastructure.id
  variable_set_id = tfe_variable_set.driftmgr_common.id
}

resource "tfe_workspace_variable_set" "common_to_environments" {
  for_each = tfe_workspace.environment_workspaces

  workspace_id    = each.value.id
  variable_set_id = tfe_variable_set.driftmgr_common.id
}

resource "tfe_workspace_variable_set" "aws_to_regional" {
  for_each = tfe_workspace.aws_regional

  workspace_id    = each.value.id
  variable_set_id = tfe_variable_set.aws_credentials.id
}

resource "tfe_workspace_variable_set" "aws_to_drift_detection" {
  workspace_id    = tfe_workspace.drift_detection.id
  variable_set_id = tfe_variable_set.aws_credentials.id
}

resource "tfe_workspace_variable_set" "azure_to_drift_detection" {
  workspace_id    = tfe_workspace.drift_detection.id
  variable_set_id = tfe_variable_set.azure_credentials.id
}

resource "tfe_workspace_variable_set" "gcp_to_drift_detection" {
  workspace_id    = tfe_workspace.drift_detection.id
  variable_set_id = tfe_variable_set.gcp_credentials.id
}

resource "tfe_workspace_variable_set" "do_to_drift_detection" {
  workspace_id    = tfe_workspace.drift_detection.id
  variable_set_id = tfe_variable_set.digitalocean_credentials.id
}

# Notification Configuration
resource "tfe_notification_configuration" "slack_infrastructure" {
  name             = "slack-infrastructure-notifications"
  enabled          = var.slack_webhook_url != ""
  destination_type = "slack"
  triggers         = ["run:completed", "run:errored", "assessment:failed"]
  url              = var.slack_webhook_url
  workspace_id     = tfe_workspace.driftmgr_infrastructure.id
}

resource "tfe_notification_configuration" "slack_drift_detection" {
  name             = "slack-drift-notifications"
  enabled          = var.slack_webhook_url != ""
  destination_type = "slack"
  triggers         = ["run:completed", "run:errored", "assessment:failed", "assessment:drifted"]
  url              = var.slack_webhook_url
  workspace_id     = tfe_workspace.drift_detection.id
}

# Run Triggers for automated drift detection
resource "tfe_run_trigger" "drift_detection_trigger" {
  workspace_id    = tfe_workspace.drift_detection.id
  sourceable_id   = tfe_workspace.driftmgr_infrastructure.id
  sourceable_type = "workspace"
}

# Teams and Team Access
resource "tfe_team" "driftmgr_admins" {
  name         = "driftmgr-admins"
  organization = data.tfe_organization.main.name
}

resource "tfe_team" "driftmgr_developers" {
  name         = "driftmgr-developers"
  organization = data.tfe_organization.main.name
}

resource "tfe_team" "driftmgr_operators" {
  name         = "driftmgr-operators"
  organization = data.tfe_organization.main.name
}

# Team access to workspaces
resource "tfe_team_access" "admins_infrastructure" {
  access       = "admin"
  team_id      = tfe_team.driftmgr_admins.id
  workspace_id = tfe_workspace.driftmgr_infrastructure.id
}

resource "tfe_team_access" "developers_infrastructure" {
  access       = "write"
  team_id      = tfe_team.driftmgr_developers.id
  workspace_id = tfe_workspace.driftmgr_infrastructure.id
}

resource "tfe_team_access" "operators_drift_detection" {
  access       = "write"
  team_id      = tfe_team.driftmgr_operators.id
  workspace_id = tfe_workspace.drift_detection.id
}

resource "tfe_team_access" "developers_environments" {
  for_each = tfe_workspace.environment_workspaces

  access       = each.key == "production" ? "read" : "write"
  team_id      = tfe_team.driftmgr_developers.id
  workspace_id = each.value.id
}

# Policy Sets for governance
resource "tfe_policy_set" "driftmgr_security" {
  name          = "driftmgr-security-policies"
  description   = "Security policies for DriftMgr infrastructure"
  organization  = data.tfe_organization.main.name
  kind          = "sentinel"
  policies_path = "policies/security"

  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.github.oauth_token_id
    branch         = "main"
  }

  workspace_ids = concat(
    [tfe_workspace.driftmgr_infrastructure.id],
    [for ws in tfe_workspace.environment_workspaces : ws.id]
  )
}

resource "tfe_policy_set" "driftmgr_cost" {
  name          = "driftmgr-cost-policies"
  description   = "Cost management policies for DriftMgr"
  organization  = data.tfe_organization.main.name
  kind          = "sentinel"
  policies_path = "policies/cost"

  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.github.oauth_token_id
    branch         = "main"
  }

  workspace_ids = [
    for ws in tfe_workspace.environment_workspaces : ws.id
    if ws.name != "driftmgr-development"
  ]
}

# Registry Modules
resource "tfe_registry_module" "driftmgr_aws" {
  vcs_repo {
    display_identifier = "${var.github_repo}/modules/aws"
    identifier         = var.github_repo
    oauth_token_id     = tfe_oauth_client.github.oauth_token_id
  }
}

resource "tfe_registry_module" "driftmgr_azure" {
  vcs_repo {
    display_identifier = "${var.github_repo}/modules/azure"
    identifier         = var.github_repo
    oauth_token_id     = tfe_oauth_client.github.oauth_token_id
  }
}

resource "tfe_registry_module" "driftmgr_gcp" {
  vcs_repo {
    display_identifier = "${var.github_repo}/modules/gcp"
    identifier         = var.github_repo
    oauth_token_id     = tfe_oauth_client.github.oauth_token_id
  }
}

# Workspace-specific variables
resource "tfe_variable" "environment_name" {
  for_each = tfe_workspace.environment_workspaces

  key          = "environment"
  value        = each.key
  category     = "terraform"
  description  = "Environment name"
  workspace_id = each.value.id
}

resource "tfe_variable" "aws_region_vars" {
  for_each = tfe_workspace.aws_regional

  key          = "aws_region"
  value        = each.key
  category     = "terraform"
  description  = "AWS region for this workspace"
  workspace_id = each.value.id
}

# Outputs
output "workspace_urls" {
  description = "URLs of created Terraform Cloud workspaces"
  value = {
    infrastructure   = "https://app.terraform.io/app/${var.tfc_organization}/workspaces/${tfe_workspace.driftmgr_infrastructure.name}"
    drift_detection  = "https://app.terraform.io/app/${var.tfc_organization}/workspaces/${tfe_workspace.drift_detection.name}"
    environments     = {
      for env, ws in tfe_workspace.environment_workspaces :
      env => "https://app.terraform.io/app/${var.tfc_organization}/workspaces/${ws.name}"
    }
    aws_regions = {
      for region, ws in tfe_workspace.aws_regional :
      region => "https://app.terraform.io/app/${var.tfc_organization}/workspaces/${ws.name}"
    }
  }
}

output "variable_set_ids" {
  description = "IDs of created variable sets"
  value = {
    common            = tfe_variable_set.driftmgr_common.id
    aws_credentials   = tfe_variable_set.aws_credentials.id
    azure_credentials = tfe_variable_set.azure_credentials.id
    gcp_credentials   = tfe_variable_set.gcp_credentials.id
    do_credentials    = tfe_variable_set.digitalocean_credentials.id
  }
}

output "team_ids" {
  description = "IDs of created teams"
  value = {
    admins     = tfe_team.driftmgr_admins.id
    developers = tfe_team.driftmgr_developers.id
    operators  = tfe_team.driftmgr_operators.id
  }
}

output "oauth_client_id" {
  description = "OAuth client ID for GitHub integration"
  value       = tfe_oauth_client.github.id
}

# Local values for organization
locals {
  workspace_tags = {
    Project     = "DriftMgr"
    ManagedBy   = "Terraform"
    Environment = "Multi"
  }
}