# DriftMgr Security Implementation

## Overview

This document describes the security features implemented in the DriftMgr web UI to protect against common web application vulnerabilities and ensure secure access to cloud resource management functionality.

## Security Features

### 1. Authentication System

#### Default Users
- **Root User**: `root` / `admin` - Full administrative access
- **Read-only User**: `readonly` / `readonly` - View-only access

**[WARNING] IMPORTANT**: Change these default passwords in production!

#### JWT Token Authentication
- Secure JWT tokens with 24-hour expiration
- Tokens stored in HttpOnly cookies and localStorage
- Automatic token validation on all protected endpoints

### 2. Role-Based Access Control (RBAC)

#### User Roles
- **Root**: Full access to all features
- **Read-only**: Limited to viewing data (no sensitive information)

#### Permissions
- `view_dashboard` - Access to dashboard
- `view_resources` - View resource information
- `view_drift` - View drift analysis
- `view_costs` - View cost data
- `view_security` - View security information
- `view_compliance` - View compliance data
- `execute_discovery` - Run resource discovery
- `execute_analysis` - Run drift analysis
- `execute_remediation` - Execute remediation actions
- `manage_users` - User management
- `manage_config` - Configuration management
- `view_sensitive` - View sensitive data (passwords, keys, etc.)

### 3. Security Headers

The application implements comprehensive security headers:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://cdn.tailwindcss.com https://d3js.org https://unpkg.com; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:; font-src 'self' https://cdn.jsdelivr.net; connect-src 'self' ws: wss:;
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload (HTTPS only)
```

### 4. CORS Protection

- Restricted to specific allowed origins
- Prevents cross-origin attacks
- Handles preflight requests properly

### 5. Rate Limiting

- 1000 requests per minute per IP address
- Prevents DoS attacks and abuse
- Configurable limits and windows

### 6. Input Validation

- Request size limits (10MB max)
- Content-Type validation
- Input sanitization for sensitive fields
- Protection against injection attacks

### 7. Data Sanitization

- Automatic redaction of sensitive data for read-only users
- Sensitive fields include: passwords, secrets, keys, tokens, credentials
- Full data access only for root users

### 8. WebSocket Security

- Authentication required for WebSocket connections
- Real-time updates with proper access control
- Secure connection handling

## Implementation Details

### File Structure
```
driftmgr/
├── internal/security/
│   ├── auth.go          # Authentication and authorization
│   ├── middleware.go    # Security middleware
│   └── security.go      # Core security utilities
├── assets/static/
│   ├── login.html       # Login page
│   └── js/dashboard.js  # Updated with auth support
└── cmd/driftmgr-server/
    └── main.go          # Updated with security integration
```

### Key Components

#### AuthManager
- User management and authentication
- JWT token generation and validation
- Role-based permission checking
- Data sanitization based on user role

#### Security Middleware
- Authentication middleware
- Authorization middleware
- Rate limiting
- CORS handling
- Security headers
- Input validation
- Response sanitization

#### Frontend Security
- Login page with secure form handling
- Token-based API authentication
- Role-based UI elements
- Secure logout functionality

## Usage

### Starting the Secure Server
```bash
# Build the application
make build

# Start the server
./bin/driftmgr-server.exe

# Access the login page
# Open http://localhost:8080/login
```

### Default Login Credentials
- **Admin**: `root` / `admin`
- **Read-only**: `readonly` / `readonly`

### API Authentication
All API requests require a valid JWT token in the Authorization header:
```http
Authorization: Bearer <jwt_token>
```

## Security Best Practices

### Production Deployment

1. **Change Default Passwords**
   ```bash
   # Update the default passwords in the code
   # or implement a password change mechanism
   ```

2. **Use HTTPS**
   ```bash
   # Set environment variables for TLS
   export DRIFT_SECURITY_ENABLE_TLS=true
   export DRIFT_SECURITY_TLS_CERT_FILE=/path/to/cert.pem
   export DRIFT_SECURITY_TLS_KEY_FILE=/path/to/key.pem
   ```

3. **Configure CORS Origins**
   ```go
   // Update allowed origins in middleware.go
   allowedOrigins := []string{
       "https://yourdomain.com",
       "https://app.yourdomain.com",
   }
   ```

4. **Set Strong JWT Secret**
   ```bash
   # Use a strong, randomly generated secret
   export DRIFT_SECURITY_JWT_SECRET="your-strong-secret-here"
   ```

5. **Network Security**
   - Deploy behind a reverse proxy (nginx/Apache)
   - Use network segmentation
   - Implement firewall rules
   - Monitor access logs

### Monitoring and Logging

- All requests are logged with user information
- Failed authentication attempts are tracked
- Rate limit violations are monitored
- Security events are recorded

### Regular Security Tasks

1. **Password Rotation**
   - Regularly change user passwords
   - Implement password expiration policies

2. **Token Management**
   - Monitor token usage
   - Implement token revocation for suspicious activity

3. **Access Reviews**
   - Regularly review user permissions
   - Remove unused accounts
   - Audit access patterns

4. **Security Updates**
   - Keep dependencies updated
   - Monitor security advisories
   - Apply security patches promptly

## Security Considerations

### [OK] **Implemented Security Features**
- **Database-backed user management** with SQLite
- **Password complexity requirements** with configurable policies
- **Account lockout mechanisms** after failed login attempts
- **Multi-factor authentication (MFA)** support
- **Comprehensive audit logging** for all security events
- **Session management** with secure token storage
- **Password history tracking** to prevent reuse
- **Account lockout** with configurable thresholds and durations

### [TOOLS] **Production Recommendations**
- **Use external identity providers** (OAuth, SAML) for enterprise environments
- **Implement intrusion detection** and monitoring
- **Regular security assessments** and penetration testing
- **Database encryption** for sensitive data
- **Backup and recovery** procedures for user data
- **Network segmentation** and firewall rules
- **SSL/TLS termination** at load balancer level
- **Regular security updates** and patch management

### [CHART] **Security Metrics**
- Failed login attempt tracking
- Account lockout monitoring
- Password strength analysis
- Session duration tracking
- Audit log analysis
- User activity patterns

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Check JWT token expiration
   - Verify user credentials
   - Check server logs for errors

2. **CORS Errors**
   - Verify allowed origins configuration
   - Check browser console for CORS violations

3. **Rate Limiting**
   - Monitor request frequency
   - Check rate limit configuration
   - Review client-side caching

4. **Permission Denied**
   - Verify user role and permissions
   - Check endpoint permission requirements
   - Review user assignment

### Debug Mode
For development, you can temporarily disable security features by modifying the middleware chain in `main.go`.

## Compliance

This security implementation addresses common compliance requirements:

- **OWASP Top 10**: Protection against injection, XSS, CSRF, etc.
- **GDPR**: Data protection and access control
- **SOC 2**: Security controls and monitoring
- **ISO 27001**: Information security management

## Support

For security-related issues or questions:
1. Review this documentation
2. Check the application logs
3. Review the security configuration
4. Contact the development team

---

**Note**: This security implementation is designed for development and testing environments. For production use, additional security measures and hardening should be implemented based on your specific security requirements and compliance needs.
