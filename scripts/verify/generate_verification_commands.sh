#!/bin/bash

# Generate provider-specific verification commands from DriftMgr output
# Usage: ./generate_verification_commands.sh [drift_results.json]

DRIFT_RESULTS=${1:-"drift_results.json"}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is required but not installed. Please install jq first."
    exit 1
fi

# Check if results file exists
if [ ! -f "$DRIFT_RESULTS" ]; then
    echo "Running DriftMgr discovery first..."
    ./driftmgr.exe discover --auto --format json --output "$DRIFT_RESULTS"
fi

echo "=== Generating Cloud Provider Verification Commands ==="
echo ""

# AWS Verification Commands
echo "### AWS CLI Verification Commands ###"
echo "# Copy and paste these into AWS CloudShell or terminal with AWS CLI configured"
echo ""

jq -r '.aws.resources[]? | 
    if .type == "aws_instance" then
        "aws ec2 describe-instances --instance-ids " + .id + " --region " + .region
    elif .type == "aws_s3_bucket" then
        "aws s3api head-bucket --bucket " + .name + " --region " + .region
    elif .type == "aws_rds_instance" then
        "aws rds describe-db-instances --db-instance-identifier " + .name + " --region " + .region
    elif .type == "aws_lambda_function" then
        "aws lambda get-function --function-name " + .name + " --region " + .region
    elif .type == "aws_security_group" then
        "aws ec2 describe-security-groups --group-ids " + .id + " --region " + .region
    elif .type == "aws_vpc" then
        "aws ec2 describe-vpcs --vpc-ids " + .id + " --region " + .region
    elif .type == "aws_subnet" then
        "aws ec2 describe-subnets --subnet-ids " + .id + " --region " + .region
    elif .type == "aws_iam_role" then
        "aws iam get-role --role-name " + .name
    elif .type == "aws_iam_user" then
        "aws iam get-user --user-name " + .name
    elif .type == "aws_dynamodb_table" then
        "aws dynamodb describe-table --table-name " + .name + " --region " + .region
    elif .type == "aws_sqs_queue" then
        "aws sqs get-queue-attributes --queue-url " + .id + " --region " + .region
    elif .type == "aws_sns_topic" then
        "aws sns get-topic-attributes --topic-arn " + .id + " --region " + .region
    elif .type == "aws_elb" or .type == "aws_lb" then
        "aws elbv2 describe-load-balancers --load-balancer-arns " + .id + " --region " + .region
    elif .type == "aws_ecs_cluster" then
        "aws ecs describe-clusters --clusters " + .name + " --region " + .region
    elif .type == "aws_ecs_service" then
        "aws ecs describe-services --cluster " + .attributes.cluster + " --services " + .name + " --region " + .region
    elif .type == "aws_ecr_repository" then
        "aws ecr describe-repositories --repository-names " + .name + " --region " + .region
    elif .type == "aws_cloudfront_distribution" then
        "aws cloudfront get-distribution --id " + .id
    elif .type == "aws_route53_zone" then
        "aws route53 get-hosted-zone --id " + .id
    elif .type == "aws_elasticache_cluster" then
        "aws elasticache describe-cache-clusters --cache-cluster-id " + .name + " --region " + .region
    else
        "# Unknown resource type: " + .type + " (ID: " + .id + ")"
    end' "$DRIFT_RESULTS" 2>/dev/null | grep -v "^null$"

echo ""
echo "### Azure CLI Verification Commands ###"
echo "# Copy and paste these into Azure Cloud Shell or terminal with Azure CLI configured"
echo ""

jq -r '.azure.resources[]? | 
    if .type == "azurerm_virtual_machine" or .type == "Microsoft.Compute/virtualMachines" then
        "az vm show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_storage_account" or .type == "Microsoft.Storage/storageAccounts" then
        "az storage account show --name " + .name
    elif .type == "azurerm_resource_group" or .type == "Microsoft.Resources/resourceGroups" then
        "az group show --name " + .name
    elif .type == "azurerm_virtual_network" or .type == "Microsoft.Network/virtualNetworks" then
        "az network vnet show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_subnet" or .type == "Microsoft.Network/virtualNetworks/subnets" then
        "az network vnet subnet show --resource-group " + .attributes.resource_group + " --vnet-name " + .attributes.vnet_name + " --name " + .name
    elif .type == "azurerm_network_security_group" or .type == "Microsoft.Network/networkSecurityGroups" then
        "az network nsg show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_public_ip" or .type == "Microsoft.Network/publicIPAddresses" then
        "az network public-ip show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_sql_server" or .type == "Microsoft.Sql/servers" then
        "az sql server show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_sql_database" or .type == "Microsoft.Sql/servers/databases" then
        "az sql db show --resource-group " + .attributes.resource_group + " --server " + .attributes.server_name + " --name " + .name
    elif .type == "azurerm_cosmosdb_account" or .type == "Microsoft.DocumentDB/databaseAccounts" then
        "az cosmosdb show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_key_vault" or .type == "Microsoft.KeyVault/vaults" then
        "az keyvault show --name " + .name
    elif .type == "azurerm_app_service_plan" or .type == "Microsoft.Web/serverfarms" then
        "az appservice plan show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_function_app" or .type == "Microsoft.Web/sites" then
        "az functionapp show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_container_registry" or .type == "Microsoft.ContainerRegistry/registries" then
        "az acr show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_kubernetes_cluster" or .type == "Microsoft.ContainerService/managedClusters" then
        "az aks show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_load_balancer" or .type == "Microsoft.Network/loadBalancers" then
        "az network lb show --resource-group " + .attributes.resource_group + " --name " + .name
    elif .type == "azurerm_redis_cache" or .type == "Microsoft.Cache/Redis" then
        "az redis show --resource-group " + .attributes.resource_group + " --name " + .name
    else
        "az resource show --ids " + .id
    end' "$DRIFT_RESULTS" 2>/dev/null | grep -v "^null$"

echo ""
echo "### GCP CLI Verification Commands ###"
echo "# Copy and paste these into Google Cloud Shell or terminal with gcloud configured"
echo ""

jq -r '.gcp.resources[]? | 
    if .type == "google_compute_instance" then
        "gcloud compute instances describe " + .name + " --zone=" + .attributes.zone + " --project=" + .attributes.project
    elif .type == "google_storage_bucket" then
        "gcloud storage buckets describe gs://" + .name + " --project=" + .attributes.project
    elif .type == "google_compute_disk" then
        "gcloud compute disks describe " + .name + " --zone=" + .attributes.zone + " --project=" + .attributes.project
    elif .type == "google_compute_network" then
        "gcloud compute networks describe " + .name + " --project=" + .attributes.project
    elif .type == "google_compute_subnetwork" then
        "gcloud compute networks subnets describe " + .name + " --region=" + .region + " --project=" + .attributes.project
    elif .type == "google_compute_firewall" then
        "gcloud compute firewall-rules describe " + .name + " --project=" + .attributes.project
    elif .type == "google_compute_address" then
        "gcloud compute addresses describe " + .name + " --region=" + .region + " --project=" + .attributes.project
    elif .type == "google_sql_database_instance" then
        "gcloud sql instances describe " + .name + " --project=" + .attributes.project
    elif .type == "google_sql_database" then
        "gcloud sql databases describe " + .name + " --instance=" + .attributes.instance + " --project=" + .attributes.project
    elif .type == "google_container_cluster" then
        "gcloud container clusters describe " + .name + " --zone=" + .attributes.zone + " --project=" + .attributes.project
    elif .type == "google_pubsub_topic" then
        "gcloud pubsub topics describe " + .name + " --project=" + .attributes.project
    elif .type == "google_pubsub_subscription" then
        "gcloud pubsub subscriptions describe " + .name + " --project=" + .attributes.project
    elif .type == "google_cloud_function" then
        "gcloud functions describe " + .name + " --region=" + .region + " --project=" + .attributes.project
    elif .type == "google_cloud_run_service" then
        "gcloud run services describe " + .name + " --region=" + .region + " --project=" + .attributes.project
    elif .type == "google_bigquery_dataset" then
        "gcloud alpha bq datasets describe " + .name + " --project=" + .attributes.project
    elif .type == "google_bigquery_table" then
        "gcloud alpha bq tables describe " + .attributes.dataset + "." + .name + " --project=" + .attributes.project
    elif .type == "google_service_account" then
        "gcloud iam service-accounts describe " + .name + " --project=" + .attributes.project
    elif .type == "google_project_iam_member" then
        "gcloud projects get-iam-policy " + .attributes.project
    else
        "# Unknown resource type: " + .type + " (ID: " + .id + ")"
    end' "$DRIFT_RESULTS" 2>/dev/null | grep -v "^null$"

echo ""
echo "### DigitalOcean CLI Verification Commands ###"
echo "# Copy and paste these into terminal with doctl configured"
echo ""

jq -r '.digitalocean.resources[]? | 
    if .type == "digitalocean_droplet" then
        "doctl compute droplet get " + .id
    elif .type == "digitalocean_volume" then
        "doctl compute volume get " + .id
    elif .type == "digitalocean_database_cluster" then
        "doctl databases get " + .id
    elif .type == "digitalocean_load_balancer" then
        "doctl compute load-balancer get " + .id
    elif .type == "digitalocean_kubernetes_cluster" then
        "doctl kubernetes cluster get " + .id
    elif .type == "digitalocean_spaces_bucket" then
        "doctl compute spaces get " + .name
    elif .type == "digitalocean_domain" then
        "doctl compute domain get " + .name
    elif .type == "digitalocean_firewall" then
        "doctl compute firewall get " + .id
    elif .type == "digitalocean_floating_ip" then
        "doctl compute floating-ip get " + .id
    elif .type == "digitalocean_vpc" then
        "doctl vpcs get " + .id
    elif .type == "digitalocean_cdn" then
        "doctl compute cdn get " + .id
    elif .type == "digitalocean_certificate" then
        "doctl compute certificate get " + .id
    elif .type == "digitalocean_ssh_key" then
        "doctl compute ssh-key get " + .id
    elif .type == "digitalocean_project" then
        "doctl projects get " + .id
    elif .type == "digitalocean_app" then
        "doctl apps get " + .id
    elif .type == "digitalocean_container_registry" then
        "doctl registry get"
    elif .type == "digitalocean_database_replica" then
        "doctl databases replica get " + .attributes.cluster_id + " " + .name
    elif .type == "digitalocean_tag" then
        "doctl compute tag get " + .name
    else
        "# Unknown resource type: " + .type + " (ID: " + .id + ")"
    end' "$DRIFT_RESULTS" 2>/dev/null | grep -v "^null$"

echo ""
echo "=== Batch Verification Script Generated ==="
echo "Save commands to separate files for batch execution:"
echo "  - aws_verify.sh"
echo "  - azure_verify.sh"
echo "  - gcp_verify.sh"
echo "  - do_verify.sh"