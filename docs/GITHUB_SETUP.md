# GitHub Repository Setup Guide for DriftMgr

This guide walks you through setting up all the necessary GitHub configurations to enable the full CI/CD pipeline and security features for DriftMgr.

## Table of Contents
- [1. GitHub Secrets Configuration](#1-github-secrets-configuration)
- [2. Branch Protection Rules](#2-branch-protection-rules)
- [3. Required Status Checks](#3-required-status-checks)
- [4. Webhook Notifications](#4-webhook-notifications)
- [5. Third-Party Integrations](#5-third-party-integrations)
- [6. Troubleshooting](#6-troubleshooting)

## 1. GitHub Secrets Configuration

Navigate to your repository → Settings → Secrets and variables → Actions

### Required Secrets

#### Essential (Must Have)
```yaml
CODECOV_TOKEN:
  description: "Token for Codecov coverage reporting"
  how_to_get: |
    1. Go to https://codecov.io/gh/catherinevee/driftmgr
    2. Click "Set up Codecov"
    3. Copy the repository upload token
  example: "12345678-1234-1234-1234-123456789abc"
```

#### Docker Hub (If Using Docker)
```yaml
DOCKER_HUB_USERNAME:
  description: "Docker Hub username"
  value: "catherinevee"

DOCKER_HUB_TOKEN:
  description: "Docker Hub access token (not password)"
  how_to_get: |
    1. Log into https://hub.docker.com
    2. Go to Account Settings → Security
    3. Create New Access Token
    4. Select "Public Repo Read/Write" permissions
  example: "dckr_pat_ABC123..."
```

#### Release Automation (Optional)
```yaml
HOMEBREW_TOKEN:
  description: "GitHub PAT for updating Homebrew formula"
  required_scopes: ["repo", "workflow"]
  how_to_get: |
    1. Go to GitHub Settings → Developer settings → Personal access tokens
    2. Generate new token (classic)
    3. Select scopes: repo, workflow
    4. Name it "Homebrew DriftMgr Updates"

SLACK_WEBHOOK_URL:
  description: "Slack webhook for release notifications"
  how_to_get: |
    1. Go to your Slack workspace
    2. Apps → Incoming Webhooks
    3. Add to Channel → Choose channel
    4. Copy webhook URL
  example: "https://hooks.slack.com/services/T00000000/B00000000/XXXX"

NOTIFICATION_EMAIL_TO:
  description: "Email address for release notifications"
  example: "team@example.com"

NOTIFICATION_EMAIL_FROM:
  description: "From email for notifications"
  example: "noreply@example.com"

SMTP_SERVER:
  description: "SMTP server for email notifications"
  example: "smtp.gmail.com"

SMTP_USERNAME:
  description: "SMTP username"
  
SMTP_PASSWORD:
  description: "SMTP password or app-specific password"
```

#### Security Scanning (Optional but Recommended)
```yaml
SNYK_TOKEN:
  description: "Snyk API token for vulnerability scanning"
  how_to_get: |
    1. Sign up at https://snyk.io
    2. Account Settings → API Token
    3. Copy token
  benefits: |
    - Enhanced vulnerability detection
    - License compliance checking
    - Dependency risk analysis
```

### How to Add Secrets

1. **Via GitHub UI:**
```bash
# Navigate to:
https://github.com/catherinevee/driftmgr/settings/secrets/actions

# Click "New repository secret"
# Enter name and value
# Click "Add secret"
```

2. **Via GitHub CLI:**
```bash
# Install GitHub CLI
brew install gh  # or download from https://cli.github.com

# Authenticate
gh auth login

# Add secrets
gh secret set CODECOV_TOKEN --body="your-token-here"
gh secret set DOCKER_HUB_USERNAME --body="catherinevee"
gh secret set DOCKER_HUB_TOKEN --body="your-docker-token"
```

3. **Via Script (Batch Add):**
```bash
#!/bin/bash
# save as add-secrets.sh

# Set your values here
CODECOV_TOKEN="your-codecov-token"
DOCKER_HUB_USERNAME="catherinevee"
DOCKER_HUB_TOKEN="your-docker-token"

# Add secrets
echo "$CODECOV_TOKEN" | gh secret set CODECOV_TOKEN
echo "$DOCKER_HUB_USERNAME" | gh secret set DOCKER_HUB_USERNAME
echo "$DOCKER_HUB_TOKEN" | gh secret set DOCKER_HUB_TOKEN

echo "✅ Secrets added successfully!"
```

## 2. Branch Protection Rules

### Main Branch Protection

Navigate to: Settings → Branches → Add rule

**Branch name pattern:** `main`

**Protection Settings:**
```yaml
✅ Require a pull request before merging
  ✅ Require approvals: 1
  ✅ Dismiss stale pull request approvals when new commits are pushed
  ✅ Require review from CODEOWNERS

✅ Require status checks to pass before merging
  ✅ Require branches to be up to date before merging
  Status checks:
    - build (ubuntu-latest, 1.23)
    - unit-tests (ubuntu-latest, 1.23)
    - gosec
    - trivy-scan
    - codeql

✅ Require conversation resolution before merging

✅ Require signed commits (optional but recommended)

✅ Require linear history

✅ Include administrators

✅ Restrict who can push to matching branches
  - Add users/teams who can push

❌ Allow force pushes (keep disabled)
❌ Allow deletions (keep disabled)
```

### Develop Branch Protection (Optional)

**Branch name pattern:** `develop`

**Protection Settings:**
```yaml
✅ Require a pull request before merging
  ✅ Require approvals: 1

✅ Require status checks to pass before merging
  Status checks:
    - build (ubuntu-latest, 1.23)
    - unit-tests (ubuntu-latest, 1.23)

✅ Include administrators
```

## 3. Required Status Checks

### Configure Status Checks

1. **Go to:** Settings → Branches → Edit rule (for main)

2. **Search and add these status checks:**

**Critical (Must Pass):**
- `build (ubuntu-latest, 1.23)` - Main build
- `unit-tests (ubuntu-latest, 1.23)` - Core tests
- `security-status` - All security scans
- `test-status` - All tests passed

**Important (Recommended):**
- `coverage-report` - Code coverage threshold
- `gosec` - Go security checker
- `trivy-scan` - Vulnerability scanner
- `codeql` - Code analysis

**Optional:**
- `build (windows-latest, 1.23)` - Windows build
- `build (macos-latest, 1.23)` - macOS build
- `integration-tests` - Integration test suite
- `race-tests` - Race condition tests

### Setting Up Auto-merge

Enable auto-merge for dependabot PRs:

1. **Repository Settings:**
```yaml
Settings → General → Features
✅ Allow auto-merge
✅ Allow Dependabot to create pull requests
```

2. **Create `.github/auto-merge.yml`:**
```yaml
# Dependabot auto-merge configuration
- match:
    dependency_type: "development"
    update_type: "semver:patch"
- match:
    dependency_type: "production"
    update_type: "security:patch"
```

## 4. Webhook Notifications

### Slack Integration

1. **Create Incoming Webhook:**
   - Go to: https://api.slack.com/apps
   - Create New App → From scratch
   - Add Incoming Webhooks feature
   - Activate and add to workspace
   - Choose channel
   - Copy webhook URL

2. **Add to GitHub Secrets:**
```bash
gh secret set SLACK_WEBHOOK_URL --body="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

3. **Test Webhook:**
```bash
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"✅ DriftMgr CI/CD Setup Complete!"}' \
  YOUR_WEBHOOK_URL
```

### Discord Integration

1. **Create Discord Webhook:**
   - Server Settings → Integrations → Webhooks
   - New Webhook → Copy URL

2. **Add GitHub Action:**
```yaml
- name: Discord Notification
  uses: sarisia/actions-status-discord@v1
  if: always()
  with:
    webhook: ${{ secrets.DISCORD_WEBHOOK }}
    status: ${{ job.status }}
    title: "Build Status"
```

### Email Notifications

1. **Gmail SMTP Setup:**
```yaml
SMTP_SERVER: smtp.gmail.com
SMTP_PORT: 587
SMTP_USERNAME: your-email@gmail.com
SMTP_PASSWORD: app-specific-password  # Not your regular password!
```

2. **Generate App Password (Gmail):**
   - Google Account → Security → 2-Step Verification
   - App passwords → Generate
   - Select "Mail"
   - Copy 16-character password

## 5. Third-Party Integrations

### Codecov Setup

1. **Sign up:** https://codecov.io
2. **Add repository:** https://app.codecov.io/gh/catherinevee/driftmgr
3. **Copy token:** Settings → General → Repository Upload Token
4. **Add to secrets:** `CODECOV_TOKEN`
5. **Badge in README:** Already added!

### Go Report Card

1. **Visit:** https://goreportcard.com/report/github.com/catherinevee/driftmgr
2. **Refresh:** Click "Refresh" to generate report
3. **Badge:** Automatically works in README

### Dependabot Configuration

Already configured in `.github/dependabot.yml`

**To enable:**
1. Settings → Security & analysis
2. Enable:
   - ✅ Dependency graph
   - ✅ Dependabot alerts
   - ✅ Dependabot security updates
   - ✅ Dependabot version updates

### GitHub Pages (for documentation)

1. **Enable Pages:**
   - Settings → Pages
   - Source: Deploy from a branch
   - Branch: main
   - Folder: /docs

2. **Custom Domain (optional):**
   - Add custom domain
   - Create CNAME file in /docs

## 6. Troubleshooting

### Common Issues and Solutions

#### Workflow Not Running
```bash
# Check workflow syntax
act --list  # Uses https://github.com/nektos/act

# Validate locally
yamllint .github/workflows/*.yml

# Check permissions
Settings → Actions → General → Workflow permissions
✅ Read and write permissions
```

#### Secret Not Working
```bash
# Verify secret exists
gh secret list

# Re-add secret (overwrites existing)
echo "new-value" | gh secret set SECRET_NAME

# Test in workflow
- run: echo "Secret length: ${#SECRET_NAME}"
  env:
    SECRET_NAME: ${{ secrets.SECRET_NAME }}
```

#### Coverage Not Updating
```bash
# Manual upload to test
bash <(curl -s https://codecov.io/bash) -t YOUR_TOKEN

# Check Codecov status
https://app.codecov.io/gh/catherinevee/driftmgr
```

#### Docker Build Failing
```bash
# Test locally
docker build -t driftmgr:test .

# Check Docker Hub limits
curl -s -H "Authorization: Bearer YOUR_TOKEN" \
  https://hub.docker.com/v2/users/catherinevee/ | jq .
```

### Monitoring CI/CD Health

1. **Actions Dashboard:**
   - https://github.com/catherinevee/driftmgr/actions

2. **Insights → Actions:**
   - View run times
   - Success/failure rates
   - Usage metrics

3. **API Status Check:**
```bash
# Get workflow runs
gh run list

# Get specific run details
gh run view RUN_ID

# Download artifacts
gh run download RUN_ID
```

### Performance Optimization

1. **Cache Dependencies:**
   - Already configured in workflows
   - Monitor cache hit rate in Actions

2. **Parallel Jobs:**
   - Matrix builds run in parallel
   - Optimize job dependencies

3. **Self-Hosted Runners (Advanced):**
```bash
# For faster builds, add self-hosted runners
Settings → Actions → Runners → New self-hosted runner
```

## Quick Setup Script

Save as `setup-github.sh`:

```bash
#!/bin/bash
set -e

echo "🚀 DriftMgr GitHub Setup Script"
echo "================================"

# Check gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI not found. Install from: https://cli.github.com"
    exit 1
fi

# Authenticate
echo "📝 Authenticating with GitHub..."
gh auth status || gh auth login

# Add essential secrets
echo "🔐 Adding secrets..."
read -p "Enter Codecov token: " CODECOV_TOKEN
echo "$CODECOV_TOKEN" | gh secret set CODECOV_TOKEN

read -p "Enter Docker Hub username [catherinevee]: " DOCKER_USER
DOCKER_USER=${DOCKER_USER:-catherinevee}
echo "$DOCKER_USER" | gh secret set DOCKER_HUB_USERNAME

read -p "Enter Docker Hub token: " DOCKER_TOKEN
echo "$DOCKER_TOKEN" | gh secret set DOCKER_HUB_TOKEN

# Enable features
echo "⚙️ Configuring repository settings..."
gh api -X PATCH /repos/catherinevee/driftmgr \
  -f has_issues=true \
  -f has_projects=false \
  -f has_wiki=false \
  -f allow_squash_merge=true \
  -f allow_merge_commit=true \
  -f allow_rebase_merge=true \
  -f delete_branch_on_merge=true

echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Configure branch protection rules manually"
echo "2. Enable Dependabot in Security settings"
echo "3. Add optional integrations (Slack, Discord, etc.)"
echo "4. Push code to trigger workflows"
```

## Verification Checklist

After setup, verify everything works:

- [ ] Secrets added to repository
- [ ] Codecov token valid
- [ ] Docker Hub credentials working
- [ ] Branch protection enabled on main
- [ ] Required status checks configured
- [ ] Dependabot enabled
- [ ] First workflow run successful
- [ ] Badges showing in README
- [ ] Notifications working (if configured)

## Support

For issues or questions:
1. Check workflow logs in Actions tab
2. Review this guide's troubleshooting section
3. Open an issue: https://github.com/catherinevee/driftmgr/issues