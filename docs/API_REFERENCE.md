# DriftMgr API Reference

## Table of Contents
- [Overview](#overview)
- [Authentication](#authentication)
- [Base URL](#base-url)
- [Rate Limiting](#rate-limiting)
- [Error Handling](#error-handling)
- [Endpoints](#endpoints)
  - [Discovery](#discovery)
  - [Drift Detection](#drift-detection)
  - [State Management](#state-management)
  - [Remediation](#remediation)
  - [Monitoring](#monitoring)
  - [Compliance](#compliance)
  - [System](#system)

## Overview

The DriftMgr API provides programmatic access to all drift detection and infrastructure management capabilities. The API follows RESTful principles and returns JSON responses.

### API Versions
- **v1**: Stable API (recommended for production)
- **v2-beta**: Beta features (subject to change)

## Authentication

### API Key Authentication
```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  https://api.driftmgr.local/v1/discovery
```

### OAuth 2.0
```bash
curl -X POST https://auth.driftmgr.local/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET"
```

Response:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

## Base URL

```
Production: https://api.driftmgr.local/v1
Staging: https://api-staging.driftmgr.local/v1
Local: http://localhost:8080/api/v1
```

## Rate Limiting

Rate limits are enforced per API key:
- **Standard**: 100 requests per minute
- **Premium**: 1000 requests per minute
- **Enterprise**: Unlimited

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "The requested resource was not found",
    "details": {
      "resource_id": "i-1234567890",
      "resource_type": "aws_instance"
    },
    "request_id": "req_123abc",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### HTTP Status Codes
- `200 OK`: Success
- `201 Created`: Resource created
- `204 No Content`: Success with no response body
- `400 Bad Request`: Invalid request
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Endpoints

### Discovery

#### List Discovered Resources
```http
GET /v1/discovery/resources
```

**Query Parameters:**
- `provider` (string): Cloud provider (aws, azure, gcp)
- `region` (string): Cloud region
- `type` (string): Resource type filter
- `tags` (object): Tag filters as JSON
- `page` (integer): Page number (default: 1)
- `limit` (integer): Items per page (default: 50, max: 100)

**Example Request:**
```bash
curl -X GET "https://api.driftmgr.local/v1/discovery/resources?provider=aws&region=us-east-1&type=ec2" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Example Response:**
```json
{
  "resources": [
    {
      "id": "i-1234567890abcdef0",
      "type": "aws_instance",
      "provider": "aws",
      "region": "us-east-1",
      "name": "web-server-01",
      "state": "running",
      "properties": {
        "instance_type": "t2.micro",
        "ami": "ami-12345678",
        "vpc_id": "vpc-12345678",
        "subnet_id": "subnet-12345678"
      },
      "tags": {
        "Environment": "production",
        "Team": "platform"
      },
      "discovered_at": "2024-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 234,
    "pages": 5
  }
}
```

#### Trigger Discovery
```http
POST /v1/discovery/scan
```

**Request Body:**
```json
{
  "providers": ["aws", "azure"],
  "regions": ["us-east-1", "us-west-2"],
  "resource_types": ["ec2", "rds", "s3"],
  "async": true
}
```

**Response:**
```json
{
  "job_id": "job_abc123",
  "status": "running",
  "started_at": "2024-01-15T10:30:00Z",
  "estimated_completion": "2024-01-15T10:35:00Z"
}
```

#### Get Discovery Job Status
```http
GET /v1/discovery/jobs/{job_id}
```

**Response:**
```json
{
  "job_id": "job_abc123",
  "status": "completed",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:34:23Z",
  "statistics": {
    "resources_discovered": 1234,
    "providers_scanned": 2,
    "regions_scanned": 4,
    "errors": 0
  },
  "results_url": "/v1/discovery/jobs/job_abc123/results"
}
```

### Drift Detection

#### Detect Drift
```http
POST /v1/drift/detect
```

**Request Body:**
```json
{
  "state_source": {
    "type": "s3",
    "bucket": "terraform-states",
    "key": "prod/terraform.tfstate"
  },
  "filters": {
    "resource_types": ["aws_instance", "aws_security_group"],
    "severity": ["critical", "high"]
  },
  "comparison_mode": "deep"
}
```

**Response:**
```json
{
  "drift_id": "drift_xyz789",
  "summary": {
    "total_resources": 150,
    "drifted_resources": 12,
    "missing_resources": 3,
    "unmanaged_resources": 5,
    "drift_percentage": 8.0
  },
  "drifts": [
    {
      "resource_id": "i-1234567890",
      "resource_type": "aws_instance",
      "drift_type": "configuration",
      "severity": "high",
      "changes": [
        {
          "property": "instance_type",
          "actual": "t2.medium",
          "expected": "t2.micro",
          "impact": "cost_increase"
        }
      ]
    }
  ],
  "detected_at": "2024-01-15T10:45:00Z"
}
```

#### Get Drift History
```http
GET /v1/drift/history
```

**Query Parameters:**
- `start_date` (ISO 8601): Start date for history
- `end_date` (ISO 8601): End date for history
- `resource_id` (string): Filter by resource ID

**Response:**
```json
{
  "history": [
    {
      "drift_id": "drift_xyz789",
      "detected_at": "2024-01-15T10:45:00Z",
      "summary": {
        "drifted_resources": 12,
        "drift_percentage": 8.0
      }
    }
  ]
}
```

### State Management

#### List State Files
```http
GET /v1/states
```

**Response:**
```json
{
  "states": [
    {
      "id": "state_123",
      "name": "production",
      "backend": "s3",
      "location": "s3://terraform-states/prod/terraform.tfstate",
      "version": 4,
      "serial": 42,
      "last_modified": "2024-01-15T09:00:00Z",
      "locked": false
    }
  ]
}
```

#### Pull State
```http
GET /v1/states/{state_id}/pull
```

**Response:**
```json
{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 42,
  "lineage": "abc-123-def",
  "resources": [
    {
      "type": "aws_instance",
      "name": "web",
      "instances": [
        {
          "attributes": {
            "id": "i-1234567890",
            "instance_type": "t2.micro"
          }
        }
      ]
    }
  ]
}
```

#### Push State
```http
POST /v1/states/{state_id}/push
```

**Request Body:**
```json
{
  "state": {
    "version": 4,
    "serial": 43,
    "resources": []
  },
  "create_backup": true,
  "force": false
}
```

#### Lock State
```http
POST /v1/states/{state_id}/lock
```

**Request Body:**
```json
{
  "operation": "drift_detection",
  "who": "user@company.com",
  "version": "1.0.0",
  "created": "2024-01-15T10:50:00Z"
}
```

**Response:**
```json
{
  "lock_id": "lock_abc123",
  "expires_at": "2024-01-15T11:50:00Z"
}
```

### Remediation

#### Generate Remediation Plan
```http
POST /v1/remediation/plan
```

**Request Body:**
```json
{
  "drift_id": "drift_xyz789",
  "strategy": "import_unmanaged",
  "target_resources": ["i-1234567890", "sg-abcdef"],
  "dry_run": true
}
```

**Response:**
```json
{
  "plan_id": "plan_456",
  "actions": [
    {
      "action": "import",
      "resource_type": "aws_instance",
      "resource_id": "i-1234567890",
      "terraform_address": "aws_instance.web[0]",
      "command": "terraform import aws_instance.web[0] i-1234567890"
    }
  ],
  "estimated_impact": {
    "resources_affected": 2,
    "risk_level": "low",
    "estimated_time": "5 minutes"
  }
}
```

#### Execute Remediation
```http
POST /v1/remediation/execute
```

**Request Body:**
```json
{
  "plan_id": "plan_456",
  "approval": {
    "approved_by": "user@company.com",
    "approved_at": "2024-01-15T11:00:00Z",
    "comment": "Approved after review"
  },
  "options": {
    "create_backup": true,
    "rollback_on_error": true,
    "parallel_execution": false
  }
}
```

### Monitoring

#### Get Monitoring Status
```http
GET /v1/monitoring/status
```

**Response:**
```json
{
  "enabled": true,
  "interval": 300,
  "last_check": "2024-01-15T10:55:00Z",
  "next_check": "2024-01-15T11:00:00Z",
  "active_monitors": [
    {
      "id": "mon_123",
      "name": "Production AWS",
      "type": "continuous",
      "status": "active",
      "resources_monitored": 150
    }
  ]
}
```

#### Create Monitor
```http
POST /v1/monitoring/monitors
```

**Request Body:**
```json
{
  "name": "Critical Resources",
  "type": "continuous",
  "interval": 60,
  "filters": {
    "tags": {
      "Environment": "production",
      "Critical": "true"
    }
  },
  "notifications": {
    "webhooks": ["https://hooks.slack.com/services/..."],
    "email": ["ops@company.com"]
  }
}
```

### Compliance

#### Run Compliance Check
```http
POST /v1/compliance/check
```

**Request Body:**
```json
{
  "standards": ["soc2", "hipaa"],
  "scope": {
    "providers": ["aws"],
    "regions": ["us-east-1"],
    "resource_types": ["s3", "rds", "ec2"]
  }
}
```

**Response:**
```json
{
  "report_id": "report_789",
  "summary": {
    "compliant": false,
    "score": 85,
    "findings": {
      "critical": 2,
      "high": 5,
      "medium": 12,
      "low": 8
    }
  },
  "findings": [
    {
      "id": "finding_001",
      "standard": "soc2",
      "control": "CC6.1",
      "severity": "critical",
      "resource": "s3://data-bucket",
      "issue": "Bucket encryption not enabled",
      "remediation": "Enable AES-256 encryption"
    }
  ]
}
```

#### Generate Compliance Report
```http
POST /v1/compliance/reports
```

**Request Body:**
```json
{
  "report_id": "report_789",
  "format": "pdf",
  "include_evidence": true,
  "executive_summary": true
}
```

**Response:**
```json
{
  "download_url": "/v1/compliance/reports/report_789/download",
  "expires_at": "2024-01-16T11:00:00Z",
  "size_bytes": 2456789,
  "format": "pdf"
}
```

### System

#### Health Check
```http
GET /v1/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "3.0.0",
  "uptime": 864000,
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "aws_api": "healthy",
    "azure_api": "degraded"
  }
}
```

#### Get Metrics
```http
GET /v1/metrics
```

**Response:**
```json
{
  "timestamp": "2024-01-15T11:00:00Z",
  "metrics": {
    "discovery": {
      "total_resources": 5678,
      "scans_today": 24,
      "avg_scan_time": 145
    },
    "drift": {
      "total_drifts": 234,
      "critical_drifts": 12,
      "avg_drift_percentage": 7.8
    },
    "api": {
      "requests_per_minute": 45,
      "avg_response_time": 123,
      "error_rate": 0.02
    }
  }
}
```

## WebSocket API

### Real-time Updates
```javascript
const ws = new WebSocket('wss://api.driftmgr.local/v1/ws');

ws.onopen = () => {
  // Subscribe to events
  ws.send(JSON.stringify({
    action: 'subscribe',
    events: ['drift_detected', 'scan_completed']
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.event, 'Data:', data.payload);
};
```

### Event Types
- `drift_detected`: New drift detected
- `scan_started`: Discovery scan started
- `scan_completed`: Discovery scan completed
- `resource_changed`: Resource configuration changed
- `compliance_violation`: Compliance violation detected
- `monitor_alert`: Monitoring alert triggered

## SDK Examples

### Python SDK
```python
from driftmgr import DriftMgrClient

client = DriftMgrClient(
    api_key='YOUR_API_KEY',
    base_url='https://api.driftmgr.local/v1'
)

# Discover resources
resources = client.discovery.list_resources(
    provider='aws',
    region='us-east-1'
)

# Detect drift
drift_report = client.drift.detect(
    state_bucket='terraform-states',
    state_key='prod/terraform.tfstate'
)

# Generate remediation plan
plan = client.remediation.create_plan(
    drift_id=drift_report.drift_id,
    strategy='import_unmanaged'
)
```

### Go SDK
```go
package main

import (
    "github.com/catherinevee/driftmgr-sdk-go"
)

func main() {
    client := driftmgr.NewClient(
        driftmgr.WithAPIKey("YOUR_API_KEY"),
        driftmgr.WithBaseURL("https://api.driftmgr.local/v1"),
    )
    
    // Discover resources
    resources, err := client.Discovery.ListResources(&driftmgr.ListResourcesInput{
        Provider: "aws",
        Region:   "us-east-1",
    })
    
    // Detect drift
    drift, err := client.Drift.Detect(&driftmgr.DetectDriftInput{
        StateBucket: "terraform-states",
        StateKey:    "prod/terraform.tfstate",
    })
}
```

### JavaScript/TypeScript SDK
```typescript
import { DriftMgrClient } from '@driftmgr/sdk';

const client = new DriftMgrClient({
  apiKey: 'YOUR_API_KEY',
  baseUrl: 'https://api.driftmgr.local/v1'
});

// Discover resources
const resources = await client.discovery.listResources({
  provider: 'aws',
  region: 'us-east-1'
});

// Detect drift
const driftReport = await client.drift.detect({
  stateBucket: 'terraform-states',
  stateKey: 'prod/terraform.tfstate'
});

// Real-time monitoring
client.monitoring.on('drift_detected', (event) => {
  console.log('Drift detected:', event);
});
```

## Postman Collection

Download the Postman collection: [DriftMgr API.postman_collection.json](./postman/DriftMgr_API.postman_collection.json)

## API Limits

| Plan       | Rate Limit | Max Resources | Parallel Scans | Data Retention |
|------------|------------|---------------|----------------|----------------|
| Free       | 10 req/min | 100           | 1              | 7 days         |
| Standard   | 100 req/min| 1,000         | 5              | 30 days        |
| Premium    | 1,000 req/min| 10,000      | 20             | 90 days        |
| Enterprise | Unlimited  | Unlimited     | Unlimited      | Custom         |

## Support

- API Status: https://status.driftmgr.local
- Documentation: https://docs.driftmgr.local
- Support: support@driftmgr.local
- GitHub: https://github.com/catherinevee/driftmgr