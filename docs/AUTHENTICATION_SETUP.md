# Authentication Setup Guide

This guide explains how to set up and use the DriftMgr authentication system with OAuth2 and API key support.

## Overview

DriftMgr supports multiple authentication methods:
- **JWT Authentication**: Username/password login with JWT tokens
- **OAuth2 Authentication**: Login with Google, GitHub, or Microsoft
- **API Key Authentication**: Programmatic access using API keys

## Environment Configuration

### OAuth2 Providers

Set up the following environment variables for OAuth2 providers:

```bash
# Google OAuth2
export GOOGLE_CLIENT_ID="your-google-client-id"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"
export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/oauth2/google/callback"

# GitHub OAuth2
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_CLIENT_SECRET="your-github-client-secret"
export GITHUB_REDIRECT_URL="http://localhost:8080/auth/oauth2/github/callback"

# Microsoft OAuth2
export MICROSOFT_CLIENT_ID="your-microsoft-client-id"
export MICROSOFT_CLIENT_SECRET="your-microsoft-client-secret"
export MICROSOFT_REDIRECT_URL="http://localhost:8080/auth/oauth2/microsoft/callback"
```

### JWT Configuration

```bash
# JWT Secret (use a strong, random secret)
export JWT_SECRET="your-super-secret-jwt-key"

# Token expiration (in seconds)
export JWT_ACCESS_TOKEN_EXPIRY="3600"  # 1 hour
export JWT_REFRESH_TOKEN_EXPIRY="604800"  # 7 days
```

## OAuth2 Setup

### 1. Google OAuth2 Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API
4. Go to "Credentials" and create OAuth 2.0 Client ID
5. Set authorized redirect URIs to your callback URL
6. Copy the Client ID and Client Secret to your environment variables

### 2. GitHub OAuth2 Setup

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Set Authorization callback URL to your callback URL
4. Copy the Client ID and Client Secret to your environment variables

### 3. Microsoft OAuth2 Setup

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to "Azure Active Directory" > "App registrations"
3. Click "New registration"
4. Set redirect URI to your callback URL
5. Copy the Application (client) ID and create a client secret
6. Set the environment variables

## API Endpoints

### Authentication Endpoints

#### JWT Authentication

```bash
# Login
POST /api/v1/auth/login
{
  "username": "admin",
  "password": "admin123"
}

# Register
POST /api/v1/auth/register
{
  "username": "newuser",
  "email": "user@example.com",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe"
}

# Refresh Token
POST /api/v1/auth/refresh
{
  "refresh_token": "your-refresh-token"
}

# Logout
POST /api/v1/auth/logout
Authorization: Bearer your-access-token
```

#### OAuth2 Authentication

```bash
# Get available OAuth2 providers
GET /api/v1/auth/oauth2/providers

# Start OAuth2 flow (redirects to provider)
GET /api/v1/auth/oauth2/google
GET /api/v1/auth/oauth2/github
GET /api/v1/auth/oauth2/microsoft

# OAuth2 callback (handled automatically)
GET /api/v1/auth/oauth2/{provider}/callback
```

#### API Key Management

```bash
# Create API Key
POST /api/v1/auth/api-keys
Authorization: Bearer your-access-token
{
  "name": "My API Key",
  "permissions": ["backend:read", "resource:read"],
  "expires_at": "2024-12-31T23:59:59Z"
}

# List API Keys
GET /api/v1/auth/api-keys
Authorization: Bearer your-access-token

# Delete API Key
DELETE /api/v1/auth/api-keys/{id}
Authorization: Bearer your-access-token
```

### Using API Keys

```bash
# Use API key in requests
curl -H "X-API-Key: your-api-key" \
     -H "Content-Type: application/json" \
     http://localhost:8080/api/v1/backends
```

## User Management

### Default Admin User

DriftMgr comes with a default admin user:
- **Username**: `admin`
- **Password**: `admin123`
- **Email**: `admin@driftmgr.com`

**Important**: Change the default password in production!

### User Roles

DriftMgr supports the following roles:

- **admin**: Full access to all features
- **operator**: Read/write access to backends, states, resources, and drift
- **viewer**: Read-only access to all features
- **auditor**: Read access with audit capabilities

### Permissions

Each role has specific permissions:

```go
// User permissions
PermissionUserRead   = "user:read"
PermissionUserWrite  = "user:write"
PermissionUserDelete = "user:delete"
PermissionUserAdmin  = "user:admin"

// Backend permissions
PermissionBackendRead   = "backend:read"
PermissionBackendWrite  = "backend:write"
PermissionBackendDelete = "backend:delete"
PermissionBackendAdmin  = "backend:admin"

// State permissions
PermissionStateRead   = "state:read"
PermissionStateWrite  = "state:write"
PermissionStateDelete = "state:delete"
PermissionStateAdmin  = "state:admin"

// Resource permissions
PermissionResourceRead   = "resource:read"
PermissionResourceWrite  = "resource:write"
PermissionResourceDelete = "resource:delete"
PermissionResourceAdmin  = "resource:admin"

// Drift permissions
PermissionDriftRead   = "drift:read"
PermissionDriftWrite  = "drift:write"
PermissionDriftDelete = "drift:delete"
PermissionDriftAdmin  = "drift:admin"
```

## Middleware Usage

### Protecting Routes

```go
// Require authentication
router.GET("/api/v1/protected", authMiddleware.RequireAuth(handler))

// Require specific permission
router.GET("/api/v1/admin", authMiddleware.RequirePermission("user:admin")(handler))

// Require specific role
router.GET("/api/v1/operator", authMiddleware.RequireRole("operator")(handler))

// Require admin privileges
router.GET("/api/v1/admin-only", authMiddleware.RequireAdmin(handler))

// API key authentication
router.GET("/api/v1/api-protected", authMiddleware.APIKeyAuth(handler))

// API key with specific permission
router.GET("/api/v1/api-read", authMiddleware.RequireAPIPermission("resource:read")(handler))
```

### Optional Authentication

```go
// Optional authentication (adds user context if authenticated)
router.GET("/api/v1/public", authMiddleware.OptionalAuth(handler))
```

## Security Best Practices

### 1. Environment Variables

- Never commit secrets to version control
- Use strong, random secrets for JWT
- Rotate secrets regularly
- Use different secrets for different environments

### 2. API Keys

- Use descriptive names for API keys
- Set appropriate expiration dates
- Grant minimal required permissions
- Rotate API keys regularly
- Monitor API key usage

### 3. OAuth2

- Use HTTPS in production
- Validate state parameters for CSRF protection
- Implement proper error handling
- Monitor OAuth2 flows for suspicious activity

### 4. JWT Tokens

- Use short expiration times for access tokens
- Implement refresh token rotation
- Store refresh tokens securely
- Validate token signatures

## Example Usage

### Complete Authentication Flow

```bash
# 1. Get available OAuth2 providers
curl http://localhost:8080/api/v1/auth/oauth2/providers

# 2. Start OAuth2 flow (redirects to Google)
curl -L http://localhost:8080/api/v1/auth/oauth2/google

# 3. After OAuth2 callback, you'll receive JWT tokens
{
  "success": true,
  "data": {
    "user": {
      "id": "user-id",
      "username": "user@example.com",
      "email": "user@example.com",
      "first_name": "John",
      "last_name": "Doe"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "token_type": "Bearer"
  }
}

# 4. Use access token for authenticated requests
curl -H "Authorization: Bearer your-access-token" \
     http://localhost:8080/api/v1/auth/profile

# 5. Create API key for programmatic access
curl -X POST \
     -H "Authorization: Bearer your-access-token" \
     -H "Content-Type: application/json" \
     -d '{"name": "My API Key", "permissions": ["backend:read"]}' \
     http://localhost:8080/api/v1/auth/api-keys

# 6. Use API key for requests
curl -H "X-API-Key: your-api-key" \
     http://localhost:8080/api/v1/backends
```

## Troubleshooting

### Common Issues

1. **OAuth2 redirect mismatch**: Ensure redirect URLs match exactly in provider settings
2. **Invalid JWT secret**: Use a strong, random secret for JWT signing
3. **API key not working**: Check permissions and expiration date
4. **CORS issues**: Configure CORS middleware properly for web applications

### Debug Mode

Enable debug logging to troubleshoot authentication issues:

```bash
export DRIFTMGR_LOG_LEVEL=debug
```

## Production Deployment

### Security Checklist

- [ ] Change default admin password
- [ ] Use strong JWT secrets
- [ ] Enable HTTPS
- [ ] Configure proper CORS settings
- [ ] Set up rate limiting
- [ ] Monitor authentication logs
- [ ] Implement proper session management
- [ ] Use secure cookie settings
- [ ] Validate all inputs
- [ ] Implement proper error handling

### Performance Considerations

- Use Redis for session storage in production
- Implement proper caching for user data
- Use connection pooling for database
- Monitor authentication performance
- Implement proper rate limiting
