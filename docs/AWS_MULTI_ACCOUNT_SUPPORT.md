# AWS Multi-Account Support in DriftMgr

DriftMgr now fully supports working with multiple AWS accounts through AWS profiles. This enhancement allows organizations to manage drift across multiple AWS accounts seamlessly.

## Features

### 1. Automatic Multi-Account Detection
DriftMgr automatically detects all configured AWS profiles and groups them by account ID. When you run `driftmgr status`, it will:
- Show the current AWS account
- List all available AWS accounts if multiple are detected
- Display which profiles belong to each account

### 2. Enhanced Credential Display

#### Single Account, Multiple Profiles
When you have multiple profiles pointing to the same AWS account:
```
AWS:            ✓ Configured
                AWS Account 025066254478
                Available profiles: default, dev, staging
```

#### Multiple Accounts
When you have profiles for different AWS accounts:
```
AWS:            ✓ Configured
                AWS Account 025066254478
                Available AWS accounts:
                  • Account 025066254478 (current)
                    Profiles: default, dev
                  • Account 987654321098
                    Profiles: prod, prod-readonly
                  • Account 123456789012
                    Profiles: sandbox
```

### 3. Account Selection with `driftmgr use aws`

The `use` command intelligently handles AWS account selection:

#### Multiple Accounts Scenario
```bash
$ driftmgr use aws

Multiple AWS Accounts Detected:
═══════════════════════════════

Account: 025066254478
───────────────────────────
1. Profile: default (Account: 025066254478)
2. Profile: dev (Account: 025066254478)

Account: 987654321098
───────────────────────────
3. Profile: prod (Account: 987654321098)
4. Profile: prod-readonly (Account: 987654321098)

Account: 123456789012
───────────────────────────
5. Profile: sandbox (Account: 123456789012)

Current profile: default

Select profile number (or press Enter to keep current): 3
✓ Switched to AWS profile: prod
```

#### Single Account Scenario
```bash
$ driftmgr use aws

Available AWS Profiles:
───────────────────────────
1. default (Account: 025066254478)
2. dev (Account: 025066254478)
3. staging (Account: 025066254478)

Current profile: default

Select profile number (or press Enter to keep current): 2
✓ Switched to AWS profile: dev
```

## Setting Up Multiple AWS Accounts

### 1. Configure AWS Profiles
Add profiles to your `~/.aws/config`:

```ini
[default]
region = us-west-2

[profile dev]
region = us-west-2
role_arn = arn:aws:iam::025066254478:role/DevRole
source_profile = default

[profile prod]
region = us-east-1
role_arn = arn:aws:iam::987654321098:role/ProdRole
source_profile = default

[profile sandbox]
region = us-west-1
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### 2. Configure Credentials
Add credentials to your `~/.aws/credentials`:

```ini
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[sandbox]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
```

## How It Works

### Account Detection Process
1. DriftMgr reads all profiles from `~/.aws/config`
2. For each profile, it attempts to call `aws sts get-caller-identity`
3. Profiles are grouped by the account ID returned
4. The current profile is determined from the `AWS_PROFILE` environment variable

### Discovery Across Accounts
When using `driftmgr discover --all-accounts`:
1. DriftMgr iterates through all detected AWS accounts
2. For each account, it uses an appropriate profile
3. Resources are discovered from each account
4. Results are aggregated and displayed

### Switching Between Accounts
```bash
# Set environment variable (temporary)
export AWS_PROFILE=prod
driftmgr discover

# Or use the built-in switcher
driftmgr use aws
# Select the prod profile

# Verify current account
driftmgr status
```

## Best Practices

### 1. Use Named Profiles
Always use descriptive profile names:
- `prod`, `prod-readonly` instead of `account1`
- `dev-us-west-2` instead of `profile1`

### 2. Organize by Environment
Group your profiles logically:
```ini
[profile dev]
[profile dev-readonly]
[profile staging]
[profile staging-readonly]
[profile prod]
[profile prod-readonly]
```

### 3. Use Role Assumption
For cross-account access, use role assumption:
```ini
[profile prod]
role_arn = arn:aws:iam::987654321098:role/DriftMgrRole
source_profile = default
mfa_serial = arn:aws:iam::025066254478:mfa/username
```

### 4. Regular Validation
Periodically validate all profiles work:
```bash
# Test all profiles
for profile in $(aws configure list-profiles); do
    echo "Testing profile: $profile"
    aws sts get-caller-identity --profile $profile
done
```

## Troubleshooting

### No Profiles Detected
- Ensure AWS CLI is installed and configured
- Check `~/.aws/config` exists and has proper format
- Verify credentials are valid with `aws sts get-caller-identity`

### Profile Not Working
- Check credentials haven't expired
- Verify role ARN is correct for assumed roles
- Ensure MFA token is provided if required

### Wrong Account Selected
- Check `AWS_PROFILE` environment variable
- Use `driftmgr use aws` to explicitly select
- Verify with `aws sts get-caller-identity --profile <name>`

## Security Considerations

1. **Never commit credentials** to version control
2. **Use IAM roles** instead of long-lived credentials where possible
3. **Enable MFA** for production accounts
4. **Use read-only roles** for discovery when possible
5. **Rotate credentials regularly**

## Example Workflow

```bash
# 1. Check current status
driftmgr status

# 2. Select production account
driftmgr use aws
# Choose prod profile

# 3. Discover resources in production
driftmgr discover

# 4. Check for drift
driftmgr drift detect

# 5. Switch to dev account
driftmgr use aws
# Choose dev profile

# 6. Compare dev resources
driftmgr discover

# 7. Generate report across all accounts
driftmgr discover --all-accounts --format json > all-accounts.json
```

## Integration with CI/CD

For automated workflows, set the profile programmatically:

```yaml
# GitHub Actions example
- name: Discover Production Resources
  env:
    AWS_PROFILE: prod
  run: |
    driftmgr discover --format json > prod-resources.json
    
- name: Discover Dev Resources  
  env:
    AWS_PROFILE: dev
  run: |
    driftmgr discover --format json > dev-resources.json
```

## Future Enhancements

- Parallel discovery across all accounts
- Cross-account drift comparison
- Account-specific drift policies
- Automated profile validation
- SSO integration support