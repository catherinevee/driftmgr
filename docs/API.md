# DriftMgr API Documentation

This document provides comprehensive documentation for the DriftMgr REST API and WebSocket API.

## 📋 Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [REST API Endpoints](#rest-api-endpoints)
- [WebSocket API](#websocket-api)
- [Examples](#examples)

## 🌟 Overview

The DriftMgr API provides programmatic access to all DriftMgr functionality through a RESTful interface and real-time WebSocket connections.

### Base URL

- **Development**: `http://localhost:8080`
- **Production**: `https://your-domain.com`

### API Version

All endpoints are prefixed with `/api/v1/`

### Content Types

- **Request**: `application/json`
- **Response**: `application/json`

## 🔐 Authentication

DriftMgr uses JWT (JSON Web Token) based authentication with optional API key support.

### JWT Authentication

Include the JWT token in the Authorization header:

```http
Authorization: Bearer <your-jwt-token>
```

### API Key Authentication

Include the API key in the X-API-Key header:

```http
X-API-Key: <your-api-key>
```

### Authentication Endpoints

#### Register User

```http
POST /api/v1/auth/register
```

**Request Body:**
```json
{
  "username": "string",
  "email": "string",
  "password": "string",
  "first_name": "string",
  "last_name": "string"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "username": "string",
    "email": "string",
    "first_name": "string",
    "last_name": "string",
    "is_active": true,
    "created_at": "2025-09-24T10:00:00Z"
  }
}
```

#### Login

```http
POST /api/v1/auth/login
```

**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "string",
    "refresh_token": "string",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

#### Refresh Token

```http
POST /api/v1/auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "string"
}
```

#### Get Profile

```http
GET /api/v1/auth/profile
```

**Headers:**
```http
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "username": "string",
    "email": "string",
    "first_name": "string",
    "last_name": "string",
    "is_admin": false,
    "created_at": "2025-09-24T10:00:00Z",
    "updated_at": "2025-09-24T10:00:00Z"
  }
}
```

## 📊 Response Format

All API responses follow a consistent format:

### Success Response

```json
{
  "success": true,
  "data": {
    // Response data
  },
  "error": null
}
```

### Error Response

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": "Additional error details"
  }
}
```

## ❌ Error Handling

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | OK - Request successful |
| 201 | Created - Resource created successfully |
| 400 | Bad Request - Invalid request data |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource not found |
| 409 | Conflict - Resource already exists |
| 422 | Unprocessable Entity - Validation failed |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error - Server error |
| 503 | Service Unavailable - Service temporarily unavailable |

### Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Request validation failed |
| `AUTHENTICATION_REQUIRED` | Authentication required |
| `INVALID_CREDENTIALS` | Invalid username or password |
| `TOKEN_EXPIRED` | JWT token has expired |
| `INSUFFICIENT_PERMISSIONS` | User lacks required permissions |
| `RESOURCE_NOT_FOUND` | Requested resource not found |
| `RESOURCE_ALREADY_EXISTS` | Resource already exists |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded |
| `INTERNAL_ERROR` | Internal server error |

## 🚦 Rate Limiting

API requests are rate limited to prevent abuse:

- **Default**: 100 requests per minute per IP
- **Authenticated**: 1000 requests per minute per user
- **Headers**: Rate limit information is included in response headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1695556800
```

## 🔌 REST API Endpoints

### Health & System

#### Health Check

```http
GET /health
GET /api/v1/health
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2025-09-24T10:00:00Z",
    "version": "1.0.0"
  }
}
```

#### Version Information

```http
GET /api/v1/version
```

**Response:**
```json
{
  "success": true,
  "data": {
    "version": "1.0.0",
    "build_time": "2025-09-24T10:00:00Z",
    "git_commit": "abc123",
    "go_version": "1.21.0"
  }
}
```

### Backend Management

#### List Backends

```http
GET /api/v1/backends/list
```

**Response:**
```json
{
  "success": true,
  "data": {
    "backends": [
      {
        "id": "uuid",
        "name": "string",
        "type": "s3",
        "config": {},
        "status": "active",
        "created_at": "2025-09-24T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

#### Discover Backends

```http
POST /api/v1/backends/discover
```

**Request Body:**
```json
{
  "providers": ["aws", "azure"],
  "regions": ["us-east-1", "us-west-2"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "discovered": 5,
    "backends": [
      {
        "id": "uuid",
        "name": "terraform-backend",
        "type": "s3",
        "config": {
          "bucket": "terraform-state",
          "region": "us-east-1"
        },
        "status": "discovered"
      }
    ]
  }
}
```

#### Get Backend Details

```http
GET /api/v1/backends/{id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "string",
    "type": "s3",
    "config": {
      "bucket": "terraform-state",
      "region": "us-east-1",
      "encrypt": true
    },
    "status": "active",
    "last_checked": "2025-09-24T10:00:00Z",
    "created_at": "2025-09-24T10:00:00Z",
    "updated_at": "2025-09-24T10:00:00Z"
  }
}
```

#### Test Backend Connection

```http
POST /api/v1/backends/{id}/test
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "success",
    "message": "Connection successful",
    "response_time": "150ms"
  }
}
```

### State Management

#### List State Files

```http
GET /api/v1/state/list
```

**Query Parameters:**
- `backend_id` (optional): Filter by backend ID
- `limit` (optional): Number of results (default: 50)
- `offset` (optional): Number of results to skip (default: 0)

**Response:**
```json
{
  "success": true,
  "data": {
    "state_files": [
      {
        "id": "uuid",
        "backend_id": "uuid",
        "path": "environments/prod/terraform.tfstate",
        "size": 1024,
        "last_modified": "2025-09-24T10:00:00Z",
        "locked": false
      }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
  }
}
```

#### Get State Details

```http
GET /api/v1/state/details
```

**Query Parameters:**
- `state_id` (required): State file ID

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "backend_id": "uuid",
    "path": "environments/prod/terraform.tfstate",
    "resources": [
      {
        "type": "aws_instance",
        "name": "web_server",
        "id": "i-1234567890abcdef0",
        "attributes": {
          "ami": "ami-12345678",
          "instance_type": "t3.micro"
        }
      }
    ],
    "outputs": {
      "instance_ip": "10.0.1.100"
    },
    "version": 1,
    "terraform_version": "1.5.0"
  }
}
```

#### Import Resource to State

```http
POST /api/v1/state/import
```

**Request Body:**
```json
{
  "state_id": "uuid",
  "resource_type": "aws_instance",
  "resource_name": "web_server",
  "resource_id": "i-1234567890abcdef0"
}
```

#### Remove Resource from State

```http
DELETE /api/v1/state/resources/{id}
```

#### Move Resource in State

```http
POST /api/v1/state/move
```

**Request Body:**
```json
{
  "state_id": "uuid",
  "from_address": "aws_instance.old_name",
  "to_address": "aws_instance.new_name"
}
```

#### Lock State File

```http
POST /api/v1/state/lock
```

**Request Body:**
```json
{
  "state_id": "uuid",
  "reason": "Running terraform apply"
}
```

#### Unlock State File

```http
POST /api/v1/state/unlock
```

**Request Body:**
```json
{
  "state_id": "uuid",
  "force": false
}
```

### Resource Management

#### List Resources

```http
GET /api/v1/resources
```

**Query Parameters:**
- `provider` (optional): Filter by cloud provider
- `type` (optional): Filter by resource type
- `region` (optional): Filter by region
- `tags` (optional): Filter by tags (comma-separated)
- `limit` (optional): Number of results (default: 50)
- `offset` (optional): Number of results to skip (default: 0)

**Response:**
```json
{
  "success": true,
  "data": {
    "resources": [
      {
        "id": "uuid",
        "provider": "aws",
        "type": "aws_instance",
        "name": "web_server",
        "region": "us-east-1",
        "status": "running",
        "tags": {
          "Environment": "production",
          "Project": "web-app"
        },
        "created_at": "2025-09-24T10:00:00Z",
        "updated_at": "2025-09-24T10:00:00Z"
      }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
  }
}
```

#### Get Resource Details

```http
GET /api/v1/resources/{id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "provider": "aws",
    "type": "aws_instance",
    "name": "web_server",
    "region": "us-east-1",
    "status": "running",
    "attributes": {
      "ami": "ami-12345678",
      "instance_type": "t3.micro",
      "vpc_security_group_ids": ["sg-12345678"],
      "subnet_id": "subnet-12345678"
    },
    "tags": {
      "Environment": "production",
      "Project": "web-app"
    },
    "cost": {
      "monthly": 15.50,
      "currency": "USD"
    },
    "compliance": {
      "status": "compliant",
      "checks": [
        {
          "name": "encryption_enabled",
          "status": "pass"
        }
      ]
    },
    "created_at": "2025-09-24T10:00:00Z",
    "updated_at": "2025-09-24T10:00:00Z"
  }
}
```

#### Search Resources

```http
GET /api/v1/resources/search
```

**Query Parameters:**
- `q` (required): Search query
- `provider` (optional): Filter by provider
- `type` (optional): Filter by type

**Response:**
```json
{
  "success": true,
  "data": {
    "resources": [
      {
        "id": "uuid",
        "provider": "aws",
        "type": "aws_instance",
        "name": "web_server",
        "region": "us-east-1",
        "status": "running",
        "tags": {
          "Environment": "production"
        }
      }
    ],
    "total": 1,
    "query": "web server"
  }
}
```

#### Update Resource Tags

```http
PUT /api/v1/resources/{id}/tags
```

**Request Body:**
```json
{
  "tags": {
    "Environment": "staging",
    "Project": "web-app",
    "Owner": "team-alpha"
  }
}
```

#### Get Resource Cost

```http
GET /api/v1/resources/{id}/cost
```

**Response:**
```json
{
  "success": true,
  "data": {
    "resource_id": "uuid",
    "cost": {
      "current_month": 15.50,
      "last_month": 14.20,
      "yearly": 186.00,
      "currency": "USD"
    },
    "breakdown": {
      "compute": 12.00,
      "storage": 2.50,
      "network": 1.00
    },
    "trend": "increasing"
  }
}
```

#### Get Resource Compliance

```http
GET /api/v1/resources/{id}/compliance
```

**Response:**
```json
{
  "success": true,
  "data": {
    "resource_id": "uuid",
    "overall_status": "compliant",
    "score": 95,
    "checks": [
      {
        "name": "encryption_enabled",
        "status": "pass",
        "description": "Resource is encrypted"
      },
      {
        "name": "backup_enabled",
        "status": "pass",
        "description": "Backup is configured"
      },
      {
        "name": "monitoring_enabled",
        "status": "fail",
        "description": "Monitoring not configured"
      }
    ],
    "last_checked": "2025-09-24T10:00:00Z"
  }
}
```

### Drift Detection

#### Detect Drift

```http
POST /api/v1/drift/detect
```

**Request Body:**
```json
{
  "resource_id": "uuid",
  "state_id": "uuid",
  "options": {
    "deep_scan": true,
    "include_deleted": false
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "uuid",
    "status": "started",
    "message": "Drift detection job started"
  }
}
```

#### List Drift Results

```http
GET /api/v1/drift/results
```

**Query Parameters:**
- `status` (optional): Filter by status (pending, running, completed, failed)
- `resource_id` (optional): Filter by resource ID
- `limit` (optional): Number of results (default: 50)
- `offset` (optional): Number of results to skip (default: 0)

**Response:**
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": "uuid",
        "job_id": "uuid",
        "resource_id": "uuid",
        "status": "completed",
        "drift_detected": true,
        "drift_count": 3,
        "created_at": "2025-09-24T10:00:00Z",
        "completed_at": "2025-09-24T10:05:00Z"
      }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
  }
}
```

#### Get Drift Result Details

```http
GET /api/v1/drift/results/{id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "job_id": "uuid",
    "resource_id": "uuid",
    "status": "completed",
    "drift_detected": true,
    "drift_count": 3,
    "drifts": [
      {
        "field": "instance_type",
        "expected": "t3.micro",
        "actual": "t3.small",
        "severity": "medium"
      },
      {
        "field": "tags.Environment",
        "expected": "production",
        "actual": "staging",
        "severity": "high"
      }
    ],
    "created_at": "2025-09-24T10:00:00Z",
    "completed_at": "2025-09-24T10:05:00Z"
  }
}
```

#### Delete Drift Result

```http
DELETE /api/v1/drift/results/{id}
```

#### Get Drift History

```http
GET /api/v1/drift/history
```

**Query Parameters:**
- `resource_id` (optional): Filter by resource ID
- `days` (optional): Number of days to look back (default: 30)

**Response:**
```json
{
  "success": true,
  "data": {
    "history": [
      {
        "date": "2025-09-24",
        "drift_count": 5,
        "resources_checked": 100
      },
      {
        "date": "2025-09-23",
        "drift_count": 3,
        "resources_checked": 95
      }
    ],
    "summary": {
      "total_drifts": 8,
      "average_daily": 4.0,
      "trend": "decreasing"
    }
  }
}
```

#### Get Drift Summary

```http
GET /api/v1/drift/summary
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_resources": 1000,
    "resources_with_drift": 25,
    "drift_percentage": 2.5,
    "critical_drifts": 5,
    "high_drifts": 10,
    "medium_drifts": 8,
    "low_drifts": 2,
    "last_scan": "2025-09-24T10:00:00Z"
  }
}
```

### WebSocket Statistics

#### Get WebSocket Stats

```http
GET /api/v1/ws/stats
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_connections": 15,
    "authenticated_users": 10,
    "anonymous_connections": 5,
    "user_connections": 8,
    "admin_connections": 2
  }
}
```

## 🔌 WebSocket API

### Connection

Connect to the WebSocket endpoint:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function(event) {
  console.log('Connected to DriftMgr WebSocket');
};

ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

ws.onclose = function(event) {
  console.log('Disconnected from DriftMgr WebSocket');
};

ws.onerror = function(error) {
  console.error('WebSocket error:', error);
};
```

### Message Format

All WebSocket messages follow this format:

```json
{
  "type": "message_type",
  "data": {
    // Message-specific data
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

### Message Types

#### Connection Established

Sent when a client successfully connects:

```json
{
  "type": "connection_established",
  "data": {
    "message": "Connected to DriftMgr WebSocket",
    "user_id": "uuid",
    "roles": ["user"]
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Drift Detection Updates

Sent when drift detection jobs are updated:

```json
{
  "type": "drift_detection",
  "data": {
    "job_id": "uuid",
    "resource_id": "uuid",
    "status": "completed",
    "drift_detected": true,
    "drift_count": 3
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Remediation Updates

Sent when remediation jobs are updated:

```json
{
  "type": "remediation_update",
  "data": {
    "job_id": "uuid",
    "resource_id": "uuid",
    "status": "running",
    "progress": 50,
    "message": "Applying remediation strategy"
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Resource Updates

Sent when resources are modified:

```json
{
  "type": "resource_update",
  "data": {
    "resource_id": "uuid",
    "action": "updated",
    "changes": {
      "tags": {
        "Environment": "staging"
      }
    }
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### State Updates

Sent when state files are modified:

```json
{
  "type": "state_update",
  "data": {
    "state_id": "uuid",
    "action": "locked",
    "reason": "Running terraform apply"
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### System Alerts

Sent for system-wide alerts:

```json
{
  "type": "system_alert",
  "data": {
    "severity": "warning",
    "message": "High drift detection rate detected",
    "details": {
      "drift_count": 25,
      "threshold": 20
    }
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Log Messages

Sent for log entries:

```json
{
  "type": "log",
  "data": {
    "level": "info",
    "message": "Drift detection completed",
    "context": {
      "job_id": "uuid",
      "resource_id": "uuid"
    }
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Heartbeat

Sent periodically to keep connections alive:

```json
{
  "type": "heartbeat",
  "data": {
    "timestamp": 1695556800,
    "server_time": "2025-09-24T10:00:00Z"
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

## 📝 Examples

### Complete Authentication Flow

```bash
# 1. Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "SecurePassword123!",
    "first_name": "Admin",
    "last_name": "User"
  }'

# 2. Login to get tokens
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "SecurePassword123!"
  }'

# 3. Use the access token for authenticated requests
curl -H "Authorization: Bearer <access-token>" \
     http://localhost:8080/api/v1/auth/profile
```

### Drift Detection Workflow

```bash
# 1. List resources
curl -H "Authorization: Bearer <token>" \
     http://localhost:8080/api/v1/resources

# 2. Start drift detection
curl -X POST http://localhost:8080/api/v1/drift/detect \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_id": "resource-uuid",
    "state_id": "state-uuid"
  }'

# 3. Check drift results
curl -H "Authorization: Bearer <token>" \
     http://localhost:8080/api/v1/drift/results
```

### WebSocket Integration

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/ws');

// Handle different message types
ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case 'connection_established':
      console.log('Connected:', message.data.message);
      break;
      
    case 'drift_detection':
      console.log('Drift detected:', message.data);
      updateDriftDisplay(message.data);
      break;
      
    case 'system_alert':
      console.log('Alert:', message.data.message);
      showAlert(message.data);
      break;
      
    case 'heartbeat':
      // Update connection status
      updateConnectionStatus('connected');
      break;
  }
};

// Update UI based on WebSocket messages
function updateDriftDisplay(data) {
  const driftElement = document.getElementById('drift-count');
  driftElement.textContent = data.drift_count;
  driftElement.className = data.drift_detected ? 'drift-detected' : 'no-drift';
}

function showAlert(data) {
  const alertContainer = document.getElementById('alerts');
  const alert = document.createElement('div');
  alert.className = `alert alert-${data.severity}`;
  alert.textContent = data.message;
  alertContainer.appendChild(alert);
}
```

### Error Handling

```javascript
// Handle API errors
async function makeAPIRequest(url, options = {}) {
  try {
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getToken()}`,
        ...options.headers
      },
      ...options
    });
    
    const data = await response.json();
    
    if (!response.ok) {
      throw new Error(data.error?.message || 'Request failed');
    }
    
    return data;
  } catch (error) {
    console.error('API request failed:', error);
    handleAPIError(error);
    throw error;
  }
}

function handleAPIError(error) {
  if (error.message.includes('Unauthorized')) {
    // Redirect to login
    window.location.href = '/login';
  } else if (error.message.includes('Forbidden')) {
    // Show permission error
    showError('You do not have permission to perform this action');
  } else {
    // Show generic error
    showError('An error occurred. Please try again.');
  }
}
```

---

This API documentation provides comprehensive coverage of all DriftMgr API endpoints and WebSocket functionality. For additional examples and use cases, refer to the main documentation or explore the API interactively using the web dashboard.
