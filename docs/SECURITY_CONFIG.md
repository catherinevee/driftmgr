# DriftMgr Security Configuration Guide

## Overview

This guide provides detailed instructions for configuring and managing the enhanced security features in DriftMgr.

## Database Configuration

### SQLite Database Setup

The default configuration uses SQLite for user management. The database is automatically created at startup.

```bash
# Set custom database path (optional)
export DRIFT_DB_PATH="/path/to/your/driftmgr.db"

# Default location: ./driftmgr.db
```

### Database Security

1. **File Permissions**: Ensure the database file has restricted permissions
   ```bash
   chmod 600 driftmgr.db
   chown driftmgr:driftmgr driftmgr.db
   ```

2. **Backup Strategy**: Implement regular database backups
   ```bash
   # Daily backup script
   cp driftmgr.db driftmgr.db.backup.$(date +%Y%m%d)
   ```

3. **Encryption**: Consider using SQLite encryption extensions for production

## Password Policy Configuration

### Default Password Policy

The system includes a comprehensive password policy with the following defaults:

```json
{
  "min_length": 8,
  "require_uppercase": true,
  "require_lowercase": true,
  "require_numbers": true,
  "require_special_chars": true,
  "max_age_days": 90,
  "prevent_reuse_count": 5,
  "lockout_threshold": 5,
  "lockout_duration_minutes": 30
}
```

### Customizing Password Policy

To modify the password policy, update the database directly:

```sql
UPDATE password_policies SET 
  min_length = 12,
  require_special_chars = true,
  max_age_days = 60,
  lockout_threshold = 3,
  lockout_duration_minutes = 60
WHERE id = 1;
```

### Password Policy Best Practices

1. **Minimum Length**: 12+ characters for production
2. **Complexity**: Require all character types
3. **History**: Prevent reuse of last 5-10 passwords
4. **Expiration**: 60-90 days for sensitive environments
5. **Lockout**: 3-5 failed attempts with 15-60 minute lockout

## Multi-Factor Authentication (MFA)

### Enabling MFA for Users

1. **Generate MFA Secret**: The system automatically generates a secret when MFA is enabled
2. **Configure TOTP App**: Use apps like Google Authenticator, Authy, or Microsoft Authenticator
3. **QR Code**: The system provides a QR code for easy setup

### MFA Configuration

```bash
# Enable MFA for a user (via API)
curl -X POST http://localhost:8080/api/v1/users/mfa/enable \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"token": "123456"}'
```

### MFA Best Practices

1. **Backup Codes**: Generate and store backup codes
2. **Device Management**: Track and manage MFA devices
3. **Recovery Process**: Establish account recovery procedures
4. **Testing**: Regularly test MFA functionality

## Account Lockout Configuration

### Lockout Settings

The system automatically locks accounts after failed login attempts:

- **Threshold**: Number of failed attempts before lockout
- **Duration**: How long the account remains locked
- **Reset**: Automatic reset on successful login

### Managing Locked Accounts

```sql
-- View locked accounts
SELECT username, failed_login_attempts, locked_until 
FROM users 
WHERE locked_until IS NOT NULL;

-- Manually unlock an account
UPDATE users 
SET failed_login_attempts = 0, locked_until = NULL 
WHERE username = 'locked_user';
```

## Audit Logging

### Audit Events

The system logs the following security events:

- User authentication (success/failure)
- Password changes
- MFA enablement/disablement
- User creation/deletion
- Permission changes
- Session management
- Administrative actions

### Audit Log Management

```sql
-- View recent audit events
SELECT user_id, action, resource, ip_address, timestamp, details
FROM audit_logs
ORDER BY timestamp DESC
LIMIT 100;

-- Clean up old audit logs (older than 1 year)
DELETE FROM audit_logs 
WHERE timestamp < datetime('now', '-1 year');
```

### Audit Log Best Practices

1. **Retention**: Keep logs for at least 1 year
2. **Monitoring**: Set up alerts for suspicious activities
3. **Backup**: Include audit logs in backup strategy
4. **Analysis**: Regular review of audit patterns

## Session Management

### Session Configuration

```bash
# JWT token expiration (default: 24 hours)
export DRIFT_JWT_EXPIRATION="24h"

# Session cleanup interval
export DRIFT_SESSION_CLEANUP_INTERVAL="1h"
```

### Session Security

1. **Token Storage**: Tokens are stored in HttpOnly cookies
2. **Secure Flag**: Cookies use Secure flag in HTTPS
3. **SameSite**: Strict SameSite policy
4. **Expiration**: Automatic token expiration
5. **Revocation**: Manual token revocation capability

## User Management

### Creating New Users

```bash
# Create a new user via API
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "password": "SecurePass123!",
    "email": "user@example.com",
    "role": "readonly"
  }'
```

### User Roles

- **root**: Full administrative access
- **readonly**: View-only access (no sensitive data)

### Password Change

```bash
# Change password via API
curl -X POST http://localhost:8080/api/v1/users/password \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "oldpassword",
    "new_password": "NewSecurePass123!"
  }'
```

## Security Headers

### Default Security Headers

The application automatically sets the following security headers:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://cdn.tailwindcss.com https://d3js.org https://unpkg.com; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:; font-src 'self' https://cdn.jsdelivr.net; connect-src 'self' ws: wss:;
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload (HTTPS only)
```

### Customizing Security Headers

Modify the security headers in `internal/security/middleware.go`:

```go
func (m *Middleware) SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Customize headers here
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        // ... other headers
        next.ServeHTTP(w, r)
    })
}
```

## Rate Limiting

### Rate Limit Configuration

```bash
# Requests per minute per IP (default: 1000)
export DRIFT_RATE_LIMIT="1000"

# Rate limit window (default: 1 minute)
export DRIFT_RATE_LIMIT_WINDOW="1m"
```

### Rate Limit Best Practices

1. **Different Limits**: Set different limits for different endpoints
2. **IP Whitelisting**: Whitelist trusted IPs
3. **Monitoring**: Monitor rate limit violations
4. **Adjustment**: Adjust limits based on usage patterns

## CORS Configuration

### CORS Settings

```go
// Update allowed origins in middleware.go
allowedOrigins := []string{
    "https://yourdomain.com",
    "https://app.yourdomain.com",
    "http://localhost:3000", // Development only
}
```

### CORS Best Practices

1. **Restrict Origins**: Only allow necessary origins
2. **Credentials**: Use `Access-Control-Allow-Credentials: true`
3. **Methods**: Limit allowed HTTP methods
4. **Headers**: Restrict allowed headers

## Monitoring and Alerting

### Security Metrics

Monitor the following security metrics:

1. **Failed Login Attempts**: Track failed authentication
2. **Account Lockouts**: Monitor locked accounts
3. **Password Changes**: Track password modifications
4. **MFA Usage**: Monitor MFA adoption
5. **Session Activity**: Track session patterns
6. **Audit Events**: Monitor security events

### Alerting Setup

```bash
# Example alerting script for failed logins
#!/bin/bash
FAILED_LOGINS=$(sqlite3 driftmgr.db "SELECT COUNT(*) FROM audit_logs WHERE action='login_failed' AND timestamp > datetime('now', '-1 hour')")
if [ $FAILED_LOGINS -gt 10 ]; then
    echo "High number of failed logins detected: $FAILED_LOGINS" | mail -s "Security Alert" admin@example.com
fi
```

## Backup and Recovery

### Database Backup

```bash
#!/bin/bash
# Daily backup script
BACKUP_DIR="/backups/driftmgr"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/driftmgr_$DATE.db"

# Create backup
cp driftmgr.db "$BACKUP_FILE"

# Compress backup
gzip "$BACKUP_FILE"

# Clean up old backups (keep 30 days)
find "$BACKUP_DIR" -name "driftmgr_*.db.gz" -mtime +30 -delete
```

### Recovery Procedures

1. **Database Recovery**: Restore from backup
2. **User Recovery**: Recreate users if necessary
3. **Audit Recovery**: Restore audit logs
4. **Configuration Recovery**: Restore security settings

## Compliance

### GDPR Compliance

1. **Data Minimization**: Only collect necessary user data
2. **Right to Erasure**: Implement user deletion
3. **Data Portability**: Export user data
4. **Consent Management**: Track user consent

### SOC 2 Compliance

1. **Access Controls**: Implement proper access management
2. **Audit Logging**: Comprehensive audit trails
3. **Change Management**: Track configuration changes
4. **Incident Response**: Document security procedures

### ISO 27001 Compliance

1. **Risk Assessment**: Regular security assessments
2. **Access Control**: Implement least privilege
3. **Cryptography**: Use strong encryption
4. **Operations Security**: Secure operational procedures

## Troubleshooting

### Common Issues

1. **Database Locked**: Check file permissions and concurrent access
2. **Authentication Failures**: Verify user credentials and lockout status
3. **MFA Issues**: Check token validity and time synchronization
4. **Session Problems**: Verify token expiration and cookie settings

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export DRIFT_DEBUG="true"
export DRIFT_LOG_LEVEL="debug"
```

### Support

For security-related issues:

1. Check the audit logs for relevant events
2. Review the security configuration
3. Test with a known good configuration
4. Contact the development team with detailed logs

---

**Note**: This configuration guide is for the enhanced security features. Always test security configurations in a development environment before applying to production.
