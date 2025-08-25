# GitHub Secrets Setup Guide

This guide explains how to set up GitHub secrets for DriftMgr's CI/CD workflows.

## Quick Start

### Method 1: Interactive Script (Recommended)

```bash
# Linux/macOS
./scripts/setup-github-secrets.sh

# Windows PowerShell
.\scripts\setup-github-secrets.ps1
```

### Method 2: GitHub CLI

```bash
# Set Docker Hub credentials
gh secret set DOCKER_HUB_USERNAME --repo=catherinevee/driftmgr
gh secret set DOCKER_HUB_TOKEN --repo=catherinevee/driftmgr
```

### Method 3: GitHub Web UI

1. Go to https://github.com/catherinevee/driftmgr/settings/secrets/actions
2. Click "New repository secret"
3. Add each secret manually

## Required Secrets

### Docker Hub (Required for Docker workflows)

| Secret Name | Description | How to Get |
|------------|-------------|------------|
| `DOCKER_HUB_USERNAME` | Your Docker Hub username | Your Docker Hub account name |
| `DOCKER_HUB_TOKEN` | Docker Hub access token | https://hub.docker.com/settings/security |

**Important**: Use an access token, not your password!

## Optional Secrets

### AWS Credentials

| Secret Name | Description |
|------------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_DEFAULT_REGION` | Default region (e.g., us-east-1) |

### Azure Credentials

| Secret Name | Description |
|------------|-------------|
| `AZURE_CLIENT_ID` | Service principal client ID |
| `AZURE_CLIENT_SECRET` | Service principal secret |
| `AZURE_TENANT_ID` | Azure tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Subscription ID |

### GCP Credentials

| Secret Name | Description |
|------------|-------------|
| `GCP_SERVICE_ACCOUNT_JSON` | Complete service account JSON |

### Other Services

| Secret Name | Description |
|------------|-------------|
| `DIGITALOCEAN_TOKEN` | DigitalOcean API token |
| `SLACK_WEBHOOK_URL` | Slack webhook for notifications |
| `TF_API_TOKEN` | Terraform Cloud API token |

## Using the Management Script

The `manage-secrets.sh` script provides comprehensive secret management:

```bash
# Interactive setup
./scripts/manage-secrets.sh setup

# Import from .env file
./scripts/manage-secrets.sh import -f .env.secrets

# Export template
./scripts/manage-secrets.sh export

# Validate configured secrets
./scripts/manage-secrets.sh validate

# Encrypt secrets file (for backup)
./scripts/manage-secrets.sh encrypt -f .env.secrets

# Decrypt secrets file
./scripts/manage-secrets.sh decrypt

# Clean up local secrets files
./scripts/manage-secrets.sh clean
```

## Bulk Import from File

1. Create a `.env.secrets` file:

```env
# .env.secrets
DOCKER_HUB_USERNAME=yourusername
DOCKER_HUB_TOKEN=dckr_pat_xxxxxxxxxxxxx
AWS_ACCESS_KEY_ID=AKIAXXXXXXXXXXXXXXXX
AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxxxxxxxxxxxxx
```

2. Import all secrets:

```bash
./scripts/manage-secrets.sh import -f .env.secrets
```

3. Clean up:

```bash
./scripts/manage-secrets.sh clean
```

## Security Best Practices

1. **Never commit secrets to Git**
   - Add `.env*` to `.gitignore`
   - Use encrypted files for backup only

2. **Use access tokens instead of passwords**
   - Docker Hub: Create token at https://hub.docker.com/settings/security
   - GitHub: Use fine-grained personal access tokens

3. **Rotate secrets regularly**
   - Update tokens every 90 days
   - Revoke unused tokens immediately

4. **Limit token permissions**
   - Docker Hub: Use read/write permissions only
   - Cloud providers: Use minimal required permissions

5. **Encrypt backup files**
   ```bash
   # Encrypt
   gpg --symmetric --cipher-algo AES256 .env.secrets
   
   # Decrypt when needed
   gpg --decrypt .env.secrets.gpg > .env.secrets
   ```

## Troubleshooting

### GitHub CLI not authenticated

```bash
gh auth login
```

### Secret not being recognized

1. Check secret name (case-sensitive)
2. Verify secret is set: `gh secret list --repo=catherinevee/driftmgr`
3. Check workflow permissions

### Docker push failing

1. Verify token has push permissions
2. Check token hasn't expired
3. Ensure username is correct

## GitHub Actions Environments

For production deployments, use GitHub Environments:

1. Go to Settings → Environments
2. Create `production` environment
3. Add required reviewers
4. Set environment-specific secrets

## Automated Setup via GitHub Actions

Run the setup workflow to get instructions:

```bash
gh workflow run setup-secrets.yml
```

This will:
- Check existing secrets
- Provide setup instructions
- Generate a template file

## Support

For issues with secrets setup:
1. Check the [GitHub Actions documentation](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
2. Review workflow logs for specific errors
3. Open an issue at https://github.com/catherinevee/driftmgr/issues