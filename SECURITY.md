# Security Policy

## Supported Versions

The following versions of DriftMgr are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| 0.9.x   | :white_check_mark: |
| < 0.9   | :x:                |

## Reporting a Vulnerability

We take the security of DriftMgr seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

1. **Do NOT** open a public issue for security vulnerabilities
2. Email details to: security@driftmgr.io (or create a GitHub Security Advisory)
3. Include the following information:
   - Type of vulnerability
   - Full paths of source file(s) related to the vulnerability
   - Location of the affected source code (tag/branch/commit or direct URL)
   - Any special configuration required to reproduce the issue
   - Step-by-step instructions to reproduce the issue
   - Proof-of-concept or exploit code (if possible)
   - Impact of the issue, including how an attacker might exploit it

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 72 hours
- **Resolution Target**:
  - Critical: 7 days
  - High: 14 days
  - Medium: 30 days
  - Low: 90 days

### What to Expect

1. We will confirm receipt of your vulnerability report
2. We will provide an initial assessment of the issue
3. We will work with you to understand and validate the issue
4. We will develop and test a fix
5. We will coordinate disclosure timing with you
6. We will publicly acknowledge your responsible disclosure (unless you prefer to remain anonymous)

## Security Best Practices

When using DriftMgr:

### Credentials Management
- Never commit credentials to version control
- Use environment variables or secure credential stores
- Rotate credentials regularly
- Use IAM roles and service accounts where possible

### Access Control
- Follow the principle of least privilege
- Use read-only credentials for discovery operations
- Separate credentials for different environments
- Enable MFA for production accounts

### Network Security
- Use TLS/SSL for all API communications
- Restrict network access to necessary endpoints only
- Use VPN or private networks for sensitive operations
- Monitor and log all access attempts

### Compliance
- Ensure compliance with relevant standards (SOC2, HIPAA, PCI-DSS)
- Regular security audits
- Keep dependencies updated
- Monitor for known vulnerabilities

## Security Features

DriftMgr includes several security features:

1. **Credential Encryption**: All stored credentials are encrypted at rest
2. **Audit Logging**: Complete audit trail of all operations
3. **RBAC Support**: Role-based access control for team environments
4. **Secrets Detection**: Built-in scanning for exposed secrets
5. **Secure Communication**: TLS/SSL for all external communications

## Known Security Limitations

- State files may contain sensitive information - handle with care
- Terraform backend credentials need appropriate access - secure accordingly
- Drift detection requires read access to cloud resources - monitor access logs

## Security Tools Integration

DriftMgr is regularly scanned with:
- Gosec (Go security checker)
- Semgrep (static analysis)
- TruffleHog (secrets detection)
- Nancy (dependency vulnerabilities)
- Snyk (vulnerability database)
- FOSSA (license compliance)

## Contact

For security concerns, contact:
- Email: security@driftmgr.io
- GitHub Security Advisories: [Report a vulnerability](https://github.com/catherinevee/driftmgr/security/advisories/new)

## Acknowledgments

We appreciate the security research community and will acknowledge researchers who responsibly disclose vulnerabilities.

---

*Last updated: December 2024*
*Policy version: 1.0*