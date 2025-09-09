# GCP Organization-Wide Drift Detection

This guide demonstrates how to perform organization-wide drift detection across all GCP projects using DriftMgr.

## Prerequisites

- GCP SDK (`gcloud`) installed and configured
- DriftMgr installed
- Service Account with Organization Viewer role
- Terraform state files in GCS buckets

## Architecture

```
         ┌──────────────────────┐
         │   Organization        │
         │  (example.com)        │
         └──────────┬───────────┘
                    │
     ┌──────────────┼──────────────┐
     │              │              │
┌────▼────┐   ┌────▼────┐   ┌────▼────┐
│ Folder: │   │ Folder: │   │ Folder: │
│  Prod   │   │   Dev   │   │  Test   │
└────┬────┘   └────┬────┘   └────┬────┘
     │              │              │
┌────▼────┐   ┌────▼────┐   ┌────▼────┐
│ Project │   │ Project │   │ Project │
│ prod-1  │   │  dev-1  │   │ test-1  │
└─────────┘   └─────────┘   └─────────┘
```

## Step 1: Service Account Setup

### Create Organization-Level Service Account

```bash
# Set variables
export ORG_ID="123456789012"
export SA_NAME="driftmgr-sa"
export PROJECT_ID="security-monitoring"

# Create service account
gcloud iam service-accounts create ${SA_NAME} \
    --display-name="DriftMgr Service Account" \
    --project=${PROJECT_ID}

# Get service account email
export SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Grant organization-level permissions
gcloud organizations add-iam-policy-binding ${ORG_ID} \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/resourcemanager.organizationViewer"

gcloud organizations add-iam-policy-binding ${ORG_ID} \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/cloudasset.viewer"

# Grant folder-level permissions (if using folders)
gcloud resource-manager folders add-iam-policy-binding FOLDER_ID \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/resourcemanager.folderViewer"

# Create and download key
gcloud iam service-accounts keys create driftmgr-key.json \
    --iam-account=${SA_EMAIL}
```

## Step 2: DriftMgr Configuration

### Configuration File (driftmgr-gcp.yaml)

```yaml
# GCP Organization Configuration
version: "1.0"

provider: gcp

# Authentication
auth:
  type: service_account
  credentials_path: ${GOOGLE_APPLICATION_CREDENTIALS}
  # Or use ADC (Application Default Credentials)
  # type: adc

# Organization settings
organization:
  id: "123456789012"
  domain: "example.com"
  
  # Folder structure (optional)
  folders:
    - name: "Production"
      id: "folders/111111111111"
      projects:
        - id: "prod-web-app"
          state_bucket: "tfstate-prod-web"
          state_prefix: "terraform/state"
        - id: "prod-data-platform"
          state_bucket: "tfstate-prod-data"
          state_prefix: "terraform/state"
    
    - name: "Development"
      id: "folders/222222222222"
      projects:
        - id: "dev-experiments"
          state_bucket: "tfstate-dev"
          state_prefix: "terraform/state"
    
    - name: "Testing"
      id: "folders/333333333333"
      projects:
        - id: "test-automation"
          state_bucket: "tfstate-test"
          state_prefix: "terraform/state"

# Discovery settings
discovery:
  # Use Asset Inventory API for faster discovery
  use_asset_inventory: true
  asset_inventory_project: "security-monitoring"
  
  # Parallel processing
  max_parallel_projects: 5
  
  # Resource filters
  resource_types:
    include:
      - "compute.googleapis.com/Instance"
      - "compute.googleapis.com/Network"
      - "compute.googleapis.com/Firewall"
      - "storage.googleapis.com/Bucket"
      - "container.googleapis.com/Cluster"
      - "iam.googleapis.com/ServiceAccount"
      - "cloudkms.googleapis.com/CryptoKey"
      - "sqladmin.googleapis.com/Instance"
      - "bigquery.googleapis.com/Dataset"
    exclude:
      - "logging.googleapis.com/*"
      - "monitoring.googleapis.com/*"
  
  # Regional settings
  regions:
    - "us-central1"
    - "us-east1"
    - "europe-west1"
    - "asia-southeast1"
  
  # Label filters
  label_filters:
    include:
      - key: "managed-by"
        value: "terraform"
    exclude:
      - key: "environment"
        value: "sandbox"

# Drift detection rules
drift_detection:
  # Ignore certain changes
  ignore_changes:
    - resource: "google_compute_instance"
      attributes:
        - "metadata.startup-script"  # Ignore startup script changes
        - "labels.last-update"        # Ignore timestamp labels
    
    - resource: "google_container_cluster"
      attributes:
        - "master_version"  # Auto-upgraded
        - "node_version"     # Auto-upgraded
  
  # Critical resources
  critical_resources:
    - type: "google_compute_firewall"
      alert_on_any_change: true
    - type: "google_iam_policy"
      alert_on_any_change: true
    - type: "google_kms_crypto_key"
      alert_on_any_change: true

# Compliance policies
compliance:
  cis_benchmark:
    enabled: true
    version: "1.3.0"
    
  custom_policies:
    - name: "Public Access Prevention"
      description: "Ensure no resources have public access"
      rules:
        - type: "storage_bucket"
          condition: "uniform_bucket_level_access.enabled == true"
        - type: "compute_instance"
          condition: "network_interface[].access_config == null"
    
    - name: "Encryption Requirements"
      description: "All storage must be encrypted"
      rules:
        - type: "storage_bucket"
          condition: "encryption.default_kms_key_name != null"
        - type: "compute_disk"
          condition: "disk_encryption_key != null"

# Notification settings
notifications:
  pubsub:
    topic: "projects/security-monitoring/topics/drift-alerts"
    
  slack:
    webhook_url: ${SLACK_WEBHOOK_URL}
    channels:
      critical: "#security-alerts"
      high: "#infrastructure-alerts"
      medium: "#devops"
  
  cloud_monitoring:
    enabled: true
    project_id: "security-monitoring"
    custom_metrics:
      - name: "drift_detection_resources_checked"
        type: "gauge"
      - name: "drift_detection_issues_found"
        type: "counter"

# Export settings
export:
  bigquery:
    enabled: true
    dataset: "drift_detection"
    table: "drift_results"
    project: "security-monitoring"
  
  gcs:
    enabled: true
    bucket: "drift-detection-reports"
    prefix: "reports/"
  
  cloud_logging:
    enabled: true
    log_name: "driftmgr"
```

## Step 3: Bash Script for Organization Scanning

```bash
#!/bin/bash
# gcp-org-drift-detection.sh

set -euo pipefail

# Configuration
CONFIG_FILE="${1:-driftmgr-gcp.yaml}"
OUTPUT_DIR="${2:-./reports}"
ORG_ID="${3:-123456789012}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Setup
echo -e "${GREEN}=== GCP Organization Drift Detection ===${NC}"
echo "Organization ID: $ORG_ID"
echo "Config File: $CONFIG_FILE"
echo "Output Directory: $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to get all projects in organization
get_all_projects() {
    gcloud projects list \
        --filter="parent.id=$ORG_ID AND parent.type=organization" \
        --format="value(projectId)"
}

# Function to get projects in folder
get_folder_projects() {
    local folder_id=$1
    gcloud projects list \
        --filter="parent.id=$folder_id AND parent.type=folder" \
        --format="value(projectId)"
}

# Function to check project drift
check_project_drift() {
    local project_id=$1
    local state_bucket=$2
    local state_prefix=$3
    
    echo -e "${YELLOW}Checking project: $project_id${NC}"
    
    # Check if state file exists
    if ! gsutil -q stat "gs://${state_bucket}/${state_prefix}/default.tfstate" 2>/dev/null; then
        echo "  No state file found, skipping..."
        return
    fi
    
    # Run drift detection
    driftmgr drift detect \
        --provider gcp \
        --project "$project_id" \
        --state-backend gcs \
        --state-bucket "$state_bucket" \
        --state-prefix "$state_prefix" \
        --config "$CONFIG_FILE" \
        --output json \
        --save-report "$OUTPUT_DIR/drift-${project_id}-$(date +%Y%m%d-%H%M%S).json" \
        2>&1 | tee "$OUTPUT_DIR/drift-${project_id}.log"
    
    # Check exit code
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "  ${GREEN}Drift detection completed${NC}"
    else
        echo -e "  ${RED}✗ Drift detection failed${NC}"
    fi
}

# Function to analyze results
analyze_results() {
    local total_drift=0
    local total_missing=0
    local critical_count=0
    
    echo -e "\n${GREEN}=== Analysis ===${NC}"
    
    for report in "$OUTPUT_DIR"/drift-*.json; do
        [ -f "$report" ] || continue
        
        project=$(basename "$report" | sed 's/drift-\(.*\)-[0-9]*.json/\1/')
        drift_count=$(jq -r '.drift_summary.drifted_resources // 0' "$report")
        missing_count=$(jq -r '.drift_summary.missing_resources // 0' "$report")
        critical=$(jq -r '.drift_summary.critical_issues // 0' "$report")
        
        total_drift=$((total_drift + drift_count))
        total_missing=$((total_missing + missing_count))
        critical_count=$((critical_count + critical))
        
        if [ "$drift_count" -gt 0 ] || [ "$missing_count" -gt 0 ]; then
            echo -e "${YELLOW}$project:${NC}"
            echo "  Drifted: $drift_count"
            echo "  Missing: $missing_count"
            [ "$critical" -gt 0 ] && echo -e "  ${RED}Critical: $critical${NC}"
        fi
    done
    
    echo -e "\n${GREEN}=== Summary ===${NC}"
    echo "Total Drifted Resources: $total_drift"
    echo "Total Missing Resources: $total_missing"
    [ "$critical_count" -gt 0 ] && echo -e "${RED}Total Critical Issues: $critical_count${NC}"
}

# Function to export to BigQuery
export_to_bigquery() {
    local dataset="drift_detection"
    local table="scan_results"
    local project="security-monitoring"
    
    echo -e "\n${GREEN}Exporting to BigQuery...${NC}"
    
    for report in "$OUTPUT_DIR"/drift-*.json; do
        [ -f "$report" ] || continue
        
        # Add metadata to report
        jq '. + {scan_timestamp: now | todate, organization_id: env.ORG_ID}' "$report" > "${report}.bq"
        
        # Load to BigQuery
        bq load \
            --source_format=NEWLINE_DELIMITED_JSON \
            --autodetect \
            "${project}:${dataset}.${table}" \
            "${report}.bq"
    done
    
    echo -e "${GREEN}Export complete${NC}"
}

# Function to send notifications
send_notifications() {
    local webhook_url="${SLACK_WEBHOOK_URL:-}"
    
    [ -z "$webhook_url" ] && return
    
    local total_drift=$(find "$OUTPUT_DIR" -name "drift-*.json" -exec jq -r '.drift_summary.drifted_resources // 0' {} \; | awk '{s+=$1} END {print s}')
    
    if [ "$total_drift" -gt 0 ]; then
        curl -X POST "$webhook_url" \
            -H 'Content-Type: application/json' \
            -d @- <<EOF
{
    "text": "🚨 *GCP Drift Detection Alert*",
    "attachments": [{
        "color": "warning",
        "fields": [
            {
                "title": "Organization",
                "value": "$ORG_ID",
                "short": true
            },
            {
                "title": "Total Drift",
                "value": "$total_drift resources",
                "short": true
            },
            {
                "title": "Scan Time",
                "value": "$(date)",
                "short": false
            }
        ]
    }]
}
EOF
    fi
}

# Main execution
main() {
    # Authenticate if needed
    if [ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]; then
        gcloud auth activate-service-account --key-file="$GOOGLE_APPLICATION_CREDENTIALS"
    fi
    
    # Get all projects
    echo -e "${GREEN}Discovering projects...${NC}"
    projects=$(get_all_projects)
    project_count=$(echo "$projects" | wc -l)
    echo "Found $project_count projects"
    echo ""
    
    # Process each project
    for project in $projects; do
        # Get state backend info (simplified - would read from config)
        state_bucket="tfstate-${project}"
        state_prefix="terraform/state"
        
        check_project_drift "$project" "$state_bucket" "$state_prefix"
    done
    
    # Analyze results
    analyze_results
    
    # Export to BigQuery
    if command -v bq &> /dev/null; then
        export_to_bigquery
    fi
    
    # Send notifications
    send_notifications
    
    echo -e "\n${GREEN}=== Drift Detection Complete ===${NC}"
    echo "Reports saved to: $OUTPUT_DIR"
}

# Run main function
main "$@"
```

## Step 4: Cloud Function for Automated Detection

```python
# main.py - Cloud Function for drift detection
import os
import json
import subprocess
from datetime import datetime
from google.cloud import storage, bigquery, pubsub_v1
from google.cloud import secretmanager
import functions_framework

@functions_framework.http
def detect_drift(request):
    """
    HTTP Cloud Function for drift detection
    Args:
        request (flask.Request): The request object
    Returns:
        The response text, or any set of values that can be turned into a Response object
    """
    
    request_json = request.get_json(silent=True)
    
    # Get parameters
    project_id = request_json.get('project_id')
    if not project_id:
        return json.dumps({'error': 'project_id is required'}), 400
    
    # Get credentials from Secret Manager
    credentials = get_credentials()
    
    # Run DriftMgr
    result = run_driftmgr(project_id, credentials)
    
    # Store results
    store_results(project_id, result)
    
    # Send notifications if drift detected
    if result.get('drift_detected'):
        send_notification(project_id, result)
    
    return json.dumps(result), 200

def run_driftmgr(project_id, credentials):
    """Run DriftMgr CLI"""
    
    # Set environment for authentication
    os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials
    
    # Run DriftMgr command
    cmd = [
        '/opt/driftmgr',
        'drift', 'detect',
        '--provider', 'gcp',
        '--project', project_id,
        '--output', 'json'
    ]
    
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=True,
            timeout=300
        )
        return json.loads(result.stdout)
    except subprocess.CalledProcessError as e:
        return {
            'error': 'DriftMgr failed',
            'message': e.stderr,
            'project_id': project_id
        }
    except subprocess.TimeoutExpired:
        return {
            'error': 'DriftMgr timeout',
            'project_id': project_id
        }

def store_results(project_id, result):
    """Store results in BigQuery"""
    
    client = bigquery.Client()
    dataset_id = 'drift_detection'
    table_id = 'results'
    
    # Prepare row
    row = {
        'project_id': project_id,
        'timestamp': datetime.utcnow().isoformat(),
        'drift_detected': result.get('drift_detected', False),
        'drift_count': result.get('drift_summary', {}).get('drifted_resources', 0),
        'missing_count': result.get('drift_summary', {}).get('missing_resources', 0),
        'full_result': json.dumps(result)
    }
    
    # Insert into BigQuery
    table = client.dataset(dataset_id).table(table_id)
    errors = client.insert_rows_json(table, [row])
    
    if errors:
        print(f'Failed to insert rows: {errors}')

def send_notification(project_id, result):
    """Send PubSub notification"""
    
    publisher = pubsub_v1.PublisherClient()
    topic_path = publisher.topic_path(
        'security-monitoring',
        'drift-alerts'
    )
    
    message = {
        'project_id': project_id,
        'timestamp': datetime.utcnow().isoformat(),
        'drift_summary': result.get('drift_summary', {}),
        'critical_resources': result.get('critical_resources', [])
    }
    
    # Publish message
    future = publisher.publish(
        topic_path,
        json.dumps(message).encode('utf-8')
    )
    future.result()

def get_credentials():
    """Get service account credentials from Secret Manager"""
    
    client = secretmanager.SecretManagerServiceClient()
    name = 'projects/security-monitoring/secrets/driftmgr-sa-key/versions/latest'
    response = client.access_secret_version(request={'name': name})
    
    # Write to temp file
    creds_file = '/tmp/credentials.json'
    with open(creds_file, 'w') as f:
        f.write(response.payload.data.decode('UTF-8'))
    
    return creds_file

# requirements.txt
"""
google-cloud-storage==2.10.0
google-cloud-bigquery==3.11.4
google-cloud-pubsub==2.18.1
google-cloud-secret-manager==2.16.3
functions-framework==3.4.0
"""
```

## Step 5: Terraform Module for Setup

```hcl
# modules/driftmgr-gcp/main.tf

variable "organization_id" {
  description = "GCP Organization ID"
  type        = string
}

variable "project_id" {
  description = "Project ID for DriftMgr resources"
  type        = string
}

variable "folders" {
  description = "List of folders to monitor"
  type = list(object({
    id   = string
    name = string
  }))
  default = []
}

# Service Account
resource "google_service_account" "driftmgr" {
  project      = var.project_id
  account_id   = "driftmgr-sa"
  display_name = "DriftMgr Service Account"
}

# Organization-level IAM
resource "google_organization_iam_member" "org_viewer" {
  org_id = var.organization_id
  role   = "roles/resourcemanager.organizationViewer"
  member = "serviceAccount:${google_service_account.driftmgr.email}"
}

resource "google_organization_iam_member" "asset_viewer" {
  org_id = var.organization_id
  role   = "roles/cloudasset.viewer"
  member = "serviceAccount:${google_service_account.driftmgr.email}"
}

# Folder-level IAM
resource "google_folder_iam_member" "folder_viewer" {
  for_each = { for f in var.folders : f.name => f }
  
  folder = each.value.id
  role   = "roles/resourcemanager.folderViewer"
  member = "serviceAccount:${google_service_account.driftmgr.email}"
}

# BigQuery Dataset for results
resource "google_bigquery_dataset" "drift_detection" {
  project    = var.project_id
  dataset_id = "drift_detection"
  location   = "US"
  
  default_table_expiration_ms = 2592000000  # 30 days
}

# BigQuery Table
resource "google_bigquery_table" "results" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.drift_detection.dataset_id
  table_id   = "results"
  
  time_partitioning {
    type  = "DAY"
    field = "timestamp"
  }
  
  schema = jsonencode([
    {
      name = "project_id"
      type = "STRING"
      mode = "REQUIRED"
    },
    {
      name = "timestamp"
      type = "TIMESTAMP"
      mode = "REQUIRED"
    },
    {
      name = "drift_detected"
      type = "BOOLEAN"
      mode = "NULLABLE"
    },
    {
      name = "drift_count"
      type = "INTEGER"
      mode = "NULLABLE"
    },
    {
      name = "missing_count"
      type = "INTEGER"
      mode = "NULLABLE"
    },
    {
      name = "full_result"
      type = "JSON"
      mode = "NULLABLE"
    }
  ])
}

# Cloud Storage Bucket for reports
resource "google_storage_bucket" "drift_reports" {
  project  = var.project_id
  name     = "${var.project_id}-drift-reports"
  location = "US"
  
  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }
  
  versioning {
    enabled = true
  }
}

# PubSub Topic for notifications
resource "google_pubsub_topic" "drift_alerts" {
  project = var.project_id
  name    = "drift-alerts"
}

# Cloud Scheduler for automated scans
resource "google_cloud_scheduler_job" "drift_scan" {
  project  = var.project_id
  name     = "drift-detection-scan"
  schedule = "0 6,18 * * *"  # Twice daily
  
  http_target {
    uri         = google_cloudfunctions_function.drift_detector.https_trigger_url
    http_method = "POST"
    
    body = base64encode(jsonencode({
      scan_all = true
    }))
    
    oidc_token {
      service_account_email = google_service_account.driftmgr.email
    }
  }
}

# Cloud Function
resource "google_cloudfunctions_function" "drift_detector" {
  project = var.project_id
  name    = "drift-detector"
  runtime = "python39"
  
  available_memory_mb   = 512
  source_archive_bucket = google_storage_bucket.functions.name
  source_archive_object = google_storage_bucket_object.function_code.name
  
  trigger_http = true
  entry_point  = "detect_drift"
  
  service_account_email = google_service_account.driftmgr.email
  
  environment_variables = {
    PROJECT_ID = var.project_id
  }
}

# Monitoring Dashboard
resource "google_monitoring_dashboard" "drift" {
  project        = var.project_id
  dashboard_json = jsonencode({
    displayName = "DriftMgr Dashboard"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          width  = 6
          height = 4
          widget = {
            title = "Drift Detection Runs"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"custom.googleapis.com/driftmgr/runs\""
                  }
                }
              }]
            }
          }
        },
        {
          xPos   = 6
          width  = 6
          height = 4
          widget = {
            title = "Drifted Resources"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"custom.googleapis.com/driftmgr/drift_count\""
                }
              }
            }
          }
        }
      ]
    }
  })
}

# Outputs
output "service_account_email" {
  value = google_service_account.driftmgr.email
}

output "bigquery_dataset" {
  value = google_bigquery_dataset.drift_detection.dataset_id
}

output "storage_bucket" {
  value = google_storage_bucket.drift_reports.name
}

output "pubsub_topic" {
  value = google_pubsub_topic.drift_alerts.id
}
```

## Best Practices

1. **Use Asset Inventory API**: For large organizations, use Cloud Asset Inventory for faster discovery
2. **Implement Rate Limiting**: Be mindful of API quotas when scanning many projects
3. **Use Workload Identity**: Prefer Workload Identity over service account keys in GKE
4. **Enable Audit Logging**: Track all DriftMgr operations in Cloud Logging
5. **Cost Optimization**: Use label filters to reduce the scope of scans
6. **Parallel Processing**: Balance between speed and API rate limits

## Integration with GCP Security Command Center

```python
# Export findings to Security Command Center
from google.cloud import securitycenter

def export_to_scc(project_id, findings):
    client = securitycenter.SecurityCenterClient()
    org_name = f"organizations/{ORG_ID}"
    source_name = f"{org_name}/sources/{SOURCE_ID}"
    
    for finding in findings:
        finding_id = f"drift-{project_id}-{finding['resource_id']}"
        finding_name = f"{source_name}/findings/{finding_id}"
        
        scc_finding = securitycenter.Finding(
            name=finding_name,
            parent=source_name,
            resource_name=f"//cloudresourcemanager.googleapis.com/projects/{project_id}",
            state=securitycenter.Finding.State.ACTIVE,
            category="DRIFT_DETECTION",
            severity=securitycenter.Finding.Severity.MEDIUM,
            finding_class=securitycenter.Finding.FindingClass.MISCONFIGURATION,
        )
        
        client.create_finding(
            request={
                "parent": source_name,
                "finding_id": finding_id,
                "finding": scc_finding,
            }
        )
```

## Next Steps

- Set up automated remediation with Cloud Build
- Integrate with Cloud Security Command Center
- Create custom Cloud Monitoring metrics
- Build Looker Studio dashboards for trends
- Implement cost analysis for drift impact

For more examples and documentation, visit [DriftMgr on GitHub](https://github.com/catherinevee/driftmgr).