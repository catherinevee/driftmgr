# Context-Sensitive Help System

## Overview

DriftMgr provides a comprehensive context-sensitive help system that offers intelligent assistance, interactive guides, and troubleshooting support throughout the application. The help system adapts to the user's current context and provides relevant information at the right time.

## Table of Contents

- [Help System Architecture](#help-system-architecture)
- [Interactive Help Features](#interactive-help-features)
- [Context-Aware Assistance](#context-aware-assistance)
- [Command Line Help](#command-line-help)
- [Web Interface Help](#web-interface-help)
- [TUI Help System](#tui-help-system)
- [Troubleshooting Guides](#troubleshooting-guides)
- [Advanced Help Features](#advanced-help-features)
- [Configuration](#configuration)
- [Developer Guide](#developer-guide)

## Help System Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────┐
│                Help System Core                         │
├─────────────────────────────────────────────────────────┤
│  Context Detection  │  Content Repository  │  Delivery  │
│  ┌─────────────────┐│ ┌─────────────────────┐│ ┌────────┐│
│  │ User State      ││ │ Help Articles       ││ │ CLI    ││
│  │ Current Command ││ │ Interactive Guides  ││ │ Web UI ││
│  │ Error Context   ││ │ Troubleshooting     ││ │ TUI    ││
│  │ Provider State  ││ │ Code Examples       ││ │ API    ││
│  └─────────────────┘│ └─────────────────────┘│ └────────┘│
└─────────────────────────────────────────────────────────┘
```

### Help Content Types

1. **Contextual Tooltips**: Brief explanations for UI elements
2. **Interactive Guides**: Step-by-step walkthroughs
3. **Troubleshooting Wizards**: Automated problem resolution
4. **Code Examples**: Working examples for common tasks
5. **Best Practices**: Recommendations and guidelines
6. **Error Resolution**: Specific help for error conditions

## Interactive Help Features

### Smart Help Suggestions

The help system proactively suggests relevant help content based on:

- Current command or operation
- Recent errors or failures
- User behavior patterns
- Configuration state
- Provider-specific context

```bash
# Example: Help suggestions during drift detection
$ driftmgr drift detect --provider aws

💡 Help Suggestion: 
   First time using AWS drift detection? Try:
   - driftmgr help drift-detection-basics
   - driftmgr tutorial aws-setup
   
   Common issues:
   - Check credentials: driftmgr validate credentials --provider aws
   - Verify regions: driftmgr help aws-regions
```

### Progressive Disclosure

Help content is organized in layers:

1. **Quick Tips**: One-line explanations
2. **Brief Help**: Paragraph-level descriptions
3. **Detailed Guides**: Comprehensive documentation
4. **Expert Mode**: Advanced configurations and troubleshooting

### Interactive Tutorials

```bash
# Start interactive tutorial
$ driftmgr tutorial

┌─ DriftMgr Interactive Tutorial ─────────────────────────┐
│                                                         │
│ Welcome to DriftMgr! Let's get you started.            │
│                                                         │
│ What would you like to learn?                           │
│                                                         │
│ 1. Basic drift detection                                │
│ 2. Multi-cloud setup                                    │
│ 3. Auto-remediation configuration                       │
│ 4. Advanced features                                     │
│                                                         │
│ Enter your choice (1-4): _                              │
└─────────────────────────────────────────────────────────┘
```

## Context-Aware Assistance

### Automatic Context Detection

The help system automatically detects context from:

```go
type HelpContext struct {
    CurrentCommand    string            `json:"current_command"`
    Provider         string            `json:"provider"`
    Operation        string            `json:"operation"`
    ErrorState       *ErrorContext     `json:"error_state,omitempty"`
    UserLevel        UserExperience    `json:"user_level"`
    LastActions      []string          `json:"last_actions"`
    ConfigState      ConfigurationState `json:"config_state"`
    EnvironmentInfo  Environment       `json:"environment"`
}
```

### Contextual Help Examples

#### During Credential Configuration

```bash
$ driftmgr config credentials --provider azure

🔐 Azure Credentials Setup
┌─────────────────────────────────────────────────┐
│ Help: Azure Authentication Methods              │
│                                                 │
│ Choose your authentication method:              │
│                                                 │
│ 1. Service Principal (Recommended for CI/CD)   │
│    • Requires: Client ID, Secret, Tenant ID    │
│    • Help: driftmgr help azure-service-principal│
│                                                 │
│ 2. Managed Identity (Azure VMs only)           │
│    • Automatic authentication                   │
│    • Help: driftmgr help azure-managed-identity │
│                                                 │
│ 3. Azure CLI (Development only)                │
│    • Uses existing Azure CLI login             │
│    • Help: driftmgr help azure-cli-auth        │
│                                                 │
│ Need help? Press 'h' for detailed guidance     │
└─────────────────────────────────────────────────┘
```

#### During Error States

```bash
$ driftmgr drift detect --provider aws
Error: AWS credentials not found

🚨 Credential Error Detected
┌─────────────────────────────────────────────────┐
│ Quick Fix: AWS Credentials Not Found            │
│                                                 │
│ Possible solutions:                             │
│                                                 │
│ 1. Set environment variables:                   │
│    export AWS_ACCESS_KEY_ID=your-key           │
│    export AWS_SECRET_ACCESS_KEY=your-secret    │
│                                                 │
│ 2. Configure AWS CLI:                          │
│    aws configure                                │
│                                                 │
│ 3. Use IAM roles (recommended):                │
│    driftmgr config aws --use-iam-role          │
│                                                 │
│ 4. Interactive setup:                          │
│    driftmgr setup aws                          │
│                                                 │
│ Try solution? [1/2/3/4/skip]: _                │
└─────────────────────────────────────────────────┘
```

## Command Line Help

### Built-in Help Commands

```bash
# Global help
driftmgr help
driftmgr --help
driftmgr -h

# Command-specific help
driftmgr help drift
driftmgr drift --help
driftmgr drift detect --help

# Topic-based help
driftmgr help aws-setup
driftmgr help troubleshooting
driftmgr help best-practices

# Interactive help
driftmgr tutorial
driftmgr wizard setup
driftmgr doctor  # System diagnostics
```

### Help Command Examples

```bash
# Basic command help
$ driftmgr help drift detect

USAGE:
    driftmgr drift detect [OPTIONS]

DESCRIPTION:
    Detect configuration drift across cloud providers by comparing
    current infrastructure state with desired configuration.

OPTIONS:
    --provider PROVIDER    Cloud provider to scan (aws, azure, gcp, digitalocean, all)
    --region REGION        Specific region to scan (can be repeated)
    --severity LEVEL       Minimum severity to report (low, medium, high, critical)
    --format FORMAT        Output format (table, json, yaml, html)
    --output FILE          Save results to file
    --parallel N           Number of parallel workers (default: 10)
    --timeout DURATION     Timeout for scan operations (default: 30m)

EXAMPLES:
    # Scan all providers
    driftmgr drift detect --provider all

    # Scan specific AWS regions
    driftmgr drift detect --provider aws --region us-east-1 --region us-west-2

    # High severity issues only
    driftmgr drift detect --severity high --format json

    # Save detailed report
    driftmgr drift detect --format html --output drift-report.html

RELATED COMMANDS:
    driftmgr drift report    # View previous drift reports
    driftmgr drift preview   # Preview changes without scanning
    driftmgr auto-remediation enable  # Enable automatic fixes

TROUBLESHOOTING:
    • No resources found: driftmgr help no-resources
    • Permission errors: driftmgr help permissions
    • Performance issues: driftmgr help performance
```

### Interactive Command Builder

```bash
$ driftmgr wizard drift-detect

┌─ Drift Detection Wizard ────────────────────────────────┐
│                                                         │
│ Let's configure your drift detection scan               │
│                                                         │
│ Step 1/5: Select Cloud Providers                       │
│                                                         │
│ ☑ AWS          ☐ Azure        ☐ GCP          ☐ DigitalOcean │
│                                                         │
│ Provider Details:                                       │
│ AWS Regions: us-east-1, us-west-2 (configure)          │
│                                                         │
│ [Next] [Help] [Cancel]                                  │
└─────────────────────────────────────────────────────────┘
```

## Web Interface Help

### Contextual Help Panel

```typescript
interface HelpPanel {
  // Always-available help sidebar
  contextualHelp: {
    currentPage: string;
    relevantTopics: HelpTopic[];
    quickActions: QuickAction[];
    searchSuggestions: string[];
  };
  
  // Inline help throughout the interface
  tooltips: {
    enabled: boolean;
    delay: number;
    showKeyboardShortcuts: boolean;
  };
  
  // Progressive disclosure
  helpLevels: {
    beginner: boolean;
    intermediate: boolean;
    expert: boolean;
  };
}
```

### Interactive Help Features

#### Smart Onboarding

```jsx
// New user onboarding flow
const OnboardingFlow = () => {
  return (
    <WelcomeTour>
      <Step target=".provider-config">
        <h3>Configure Cloud Providers</h3>
        <p>Connect DriftMgr to your cloud accounts to start monitoring.</p>
        <Actions>
          <Button variant="primary">Configure AWS</Button>
          <Button variant="secondary">Skip for now</Button>
        </Actions>
      </Step>
      
      <Step target=".drift-detection">
        <h3>Run Your First Drift Scan</h3>
        <p>Detect configuration differences in your infrastructure.</p>
        <CodeExample>
          driftmgr drift detect --provider aws
        </CodeExample>
      </Step>
    </WelcomeTour>
  );
};
```

#### Help Search

```jsx
const HelpSearch = () => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  
  return (
    <SearchBox>
      <Input 
        placeholder="Search help, commands, or ask a question..."
        value={query}
        onChange={handleSearch}
      />
      <SearchResults>
        {results.map(result => (
          <SearchResult key={result.id}>
            <Title>{result.title}</Title>
            <Snippet>{result.snippet}</Snippet>
            <Actions>
              <Link to={result.url}>Read More</Link>
              <Button onClick={() => executeExample(result.example)}>
                Try Example
              </Button>
            </Actions>
          </SearchResult>
        ))}
      </SearchResults>
    </SearchBox>
  );
};
```

### Guided Workflows

```jsx
const GuidedWorkflow = ({ workflow }) => {
  return (
    <WorkflowWizard>
      <ProgressIndicator current={currentStep} total={totalSteps} />
      
      <StepContent>
        <StepTitle>{currentStep.title}</StepTitle>
        <StepDescription>{currentStep.description}</StepDescription>
        
        {currentStep.hasForm && (
          <StepForm onSubmit={handleStepSubmit}>
            {renderFormFields(currentStep.fields)}
          </StepForm>
        )}
        
        <HelpSection>
          <ExpandableHelp title="Need Help?">
            <HelpContent>{currentStep.helpContent}</HelpContent>
            <QuickActions>
              <Button onClick={() => openTutorial(currentStep.tutorial)}>
                Watch Tutorial
              </Button>
              <Button onClick={() => openDocumentation(currentStep.docs)}>
                Read Docs
              </Button>
            </QuickActions>
          </ExpandableHelp>
        </HelpSection>
      </StepContent>
      
      <StepNavigation>
        <Button variant="secondary" onClick={previousStep}>
          Previous
        </Button>
        <Button variant="primary" onClick={nextStep}>
          Next
        </Button>
      </StepNavigation>
    </WorkflowWizard>
  );
};
```

## TUI Help System

### Built-in Help Navigation

```
┌─ DriftMgr Help ─────────────────────────────────────────┐
│                                                         │
│ Navigation:                                             │
│ ↑/↓ or j/k  - Navigate items                           │
│ Enter       - Select item                               │
│ q           - Quit help                                 │
│ /           - Search help content                       │
│ ?           - Show keyboard shortcuts                   │
│                                                         │
│ Help Categories:                                        │
│                                                         │
│ 📚 Getting Started                                      │
│   ├── Quick Start Guide                                │
│   ├── Configuration Setup                              │
│   └── First Drift Scan                                 │
│                                                         │
│ 🔧 Commands                                             │
│   ├── Drift Detection                                  │
│   ├── Auto-Remediation                                 │
│   ├── Resource Discovery                               │
│   └── Cost Analysis                                    │
│                                                         │
│ 🌐 Cloud Providers                                      │
│   ├── AWS Setup                                        │
│   ├── Azure Configuration                              │
│   ├── GCP Integration                                  │
│   └── DigitalOcean Setup                               │
│                                                         │
│ 🚨 Troubleshooting                                      │
│   ├── Common Issues                                    │
│   ├── Error Messages                                   │
│   ├── Performance Problems                             │
│   └── Credential Issues                                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Context-Sensitive Help in TUI

```
┌─ Drift Detection Results ───────────────────────────────┐
│                                                         │
│ 🔍 Scanning AWS (us-east-1)... [████████████] 100%     │
│                                                         │
│ Results: 5 drifts found                                 │
│                                                         │
│ ┌─ High Priority Drifts ────────────────────────────┐   │
│ │ [WARNING]  ec2-instance-123: Security group modified    │   │
│ │ 🔄 rds-instance-456: Parameter group changed     │   │
│ │ 📊 s3-bucket-789: Lifecycle policy updated       │   │
│ └───────────────────────────────────────────────────┘   │
│                                                         │
│ Actions:                                                │
│ [R] Remediate Selected  [V] View Details  [E] Export    │
│ [A] Auto-fix All        [S] Save Report   [H] Help      │
│                                                         │
│ Press 'H' for help with drift remediation              │
└─────────────────────────────────────────────────────────┘

# When user presses 'H':
┌─ Drift Remediation Help ────────────────────────────────┐
│                                                         │
│ 🔧 Remediation Options:                                 │
│                                                         │
│ R - Remediate Selected                                  │
│     • Fix the highlighted drift                        │
│     • Creates backup before changes                    │
│     • Shows preview of actions                         │
│                                                         │
│ A - Auto-fix All                                       │
│     • Attempts to fix all detected drifts             │
│     • Only applies low-risk remediations               │
│     • Requires confirmation for each action            │
│                                                         │
│ V - View Details                                        │
│     • Shows exact configuration differences            │
│     • Displays root cause analysis                     │
│     • Provides manual fix instructions                 │
│                                                         │
│ Safety Features:                                        │
│ • All changes create automatic backups                 │
│ • Dry-run mode available (--dry-run flag)             │
│ • Approval required for high-risk changes              │
│                                                         │
│ Related Commands:                                       │
│ • driftmgr backup list     - View available backups    │
│ • driftmgr rollback        - Undo recent changes       │
│ • driftmgr simulate        - Test fixes safely         │
│                                                         │
│ [Press any key to continue]                             │
└─────────────────────────────────────────────────────────┘
```

## Troubleshooting Guides

### Automated Problem Detection

```bash
$ driftmgr doctor

🔍 DriftMgr System Diagnostics
┌─────────────────────────────────────────────────────────┐
│                                                         │
│ [OK] Go version: 1.21.0 (OK)                             │
│ [OK] Config file: Found at ~/.driftmgr/config.yaml       │
│ [WARNING]  AWS credentials: Found but expired                  │
│ [ERROR] Azure credentials: Not configured                    │
│ [OK] Network connectivity: OK                            │
│ [WARNING]  Disk space: 89% full (consider cleanup)           │
│                                                         │
│ Issues Detected:                                        │
│                                                         │
│ 1. AWS credentials expired                              │
│    Fix: aws configure                                   │
│    Or: driftmgr config aws --refresh                   │
│                                                         │
│ 2. Azure not configured                                 │
│    Fix: driftmgr setup azure                           │
│    Guide: driftmgr help azure-setup                    │
│                                                         │
│ 3. Low disk space                                       │
│    Fix: driftmgr cleanup                               │
│    Or: rm -rf ~/.driftmgr/cache/*                      │
│                                                         │
│ Run automatic fixes? [y/N]: _                          │
└─────────────────────────────────────────────────────────┘
```

### Problem-Specific Guides

#### Credential Issues

```markdown
# Troubleshooting: AWS Credential Problems

## Symptoms
- "AWS credentials not found" error
- "Access Denied" when scanning resources
- "Invalid security token" errors

## Diagnosis
Run: `driftmgr validate credentials --provider aws --verbose`

## Common Solutions

### 1. Environment Variables Not Set
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

### 2. AWS CLI Not Configured
```bash
aws configure
# Follow prompts to enter credentials
```

### 3. Credentials Expired
```bash
# For temporary credentials
aws sts get-caller-identity

# Refresh if needed
aws configure
```

### 4. Insufficient Permissions
Required IAM permissions:
- `ec2:Describe*`
- `s3:GetBucket*`
- `rds:Describe*`
(See SECURITY.md for complete list)

## Advanced Troubleshooting

### Check Credential Chain
```bash
driftmgr debug credentials --provider aws
```

### Test Specific Permissions
```bash
driftmgr validate permissions --provider aws --service ec2
```
```

#### Performance Issues

```markdown
# Troubleshooting: Performance Problems

## Symptoms
- Slow drift detection scans
- High memory usage
- API timeout errors

## Diagnosis
```bash
driftmgr performance profile --duration 60s
```

## Common Solutions

### 1. Reduce Parallel Workers
```yaml
# config.yaml
drift_detection:
  parallel_workers: 5  # Reduce from default 10
```

### 2. Increase Timeouts
```yaml
timeouts:
  api_timeout: 60s
  scan_timeout: 30m
```

### 3. Enable Caching
```yaml
cache:
  enabled: true
  ttl: 15m
  max_size: 1GB
```

### 4. Limit Scope
```bash
# Scan specific regions only
driftmgr drift detect --provider aws --region us-east-1

# Scan specific services
driftmgr drift detect --service ec2,s3
```
```

### Interactive Troubleshooting

```bash
$ driftmgr troubleshoot

🔧 Interactive Troubleshooting
┌─────────────────────────────────────────────────────────┐
│                                                         │
│ What problem are you experiencing?                      │
│                                                         │
│ 1. Credential/authentication issues                     │
│ 2. Drift detection not working                         │
│ 3. Performance/timeout problems                        │
│ 4. Auto-remediation failures                           │
│ 5. Web interface not loading                           │
│ 6. Configuration problems                               │
│ 7. Other (describe your issue)                         │
│                                                         │
│ Enter your choice (1-7): 2                             │
│                                                         │
│ Drift Detection Troubleshooting:                       │
│                                                         │
│ Let's diagnose the issue step by step...               │
│                                                         │
│ [OK] Checking configuration file... OK                   │
│ [OK] Validating cloud credentials... OK                  │
│ [WARNING]  Testing API connectivity... TIMEOUT                │
│                                                         │
│ Issue found: API timeouts                              │
│                                                         │
│ Suggested solutions:                                    │
│ 1. Increase timeout values                             │
│ 2. Check network connectivity                          │
│ 3. Reduce scan scope                                   │
│                                                         │
│ Try solution 1? [y/N]: y                               │
│                                                         │
│ Updating timeout configuration...                      │
│ [OK] Updated config.yaml                                 │
│                                                         │
│ Test the fix? [y/N]: y                                 │
│                                                         │
│ Running test scan... [OK] SUCCESS                        │
│                                                         │
│ Problem resolved! Your drift detection should now      │
│ work properly.                                          │
└─────────────────────────────────────────────────────────┘
```

## Advanced Help Features

### AI-Powered Help Assistant

```bash
$ driftmgr ask "How do I set up auto-remediation for AWS with approval workflows?"

🤖 DriftMgr Assistant:

To set up auto-remediation for AWS with approval workflows:

1. **Enable auto-remediation:**
   ```bash
   driftmgr auto-remediation enable --provider aws
   ```

2. **Configure approval workflow:**
   ```yaml
   # configs/auto-remediation.yaml
   approval_workflow:
     enabled: true
     approval_timeout: 30m
     approvers:
       - email: "admin@company.com"
       - slack: "#ops-team"
   ```

3. **Set up rules requiring approval:**
   ```yaml
   rules:
     - name: high-risk-changes
       requires_approval: true
       approval_level: 2  # Requires 2 approvals
       max_risk_level: high
   ```

4. **Test the workflow:**
   ```bash
   driftmgr auto-remediation test --dry-run
   ```

**Related topics:**
- Auto-remediation safety controls
- Approval notification setup  
- Risk assessment configuration

**Need more help?** Try:
- `driftmgr tutorial auto-remediation`
- `driftmgr help approval-workflows`
```

### Smart Error Resolution

```bash
$ driftmgr drift detect --provider azure
Error: AADSTS70011: The provided value for the input parameter 'scope' is not valid.

🤖 Smart Error Resolution:

This Azure authentication error suggests a scope configuration issue.

**Likely causes:**
1. Incorrect Azure application registration
2. Missing API permissions
3. Wrong tenant ID configuration

**Recommended fixes:**

**Quick fix:**
```bash
driftmgr config azure --reset-auth
driftmgr setup azure  # Re-run setup wizard
```

**Manual fix:**
1. Check your Azure app registration:
   - Go to Azure Portal > App Registrations
   - Verify your app has required permissions
   - Ensure admin consent is granted

2. Update your configuration:
   ```yaml
   providers:
     azure:
       tenant_id: "your-correct-tenant-id"
       client_id: "your-app-client-id"
   ```

**Apply recommended fix?** [y/N]: y

Applying quick fix...
[OK] Reset Azure authentication
[OK] Started setup wizard

Please follow the setup wizard to reconfigure Azure.
```

### Learning Path Recommendations

```bash
$ driftmgr learn

📚 DriftMgr Learning Paths
┌─────────────────────────────────────────────────────────┐
│                                                         │
│ Based on your usage, we recommend:                      │
│                                                         │
│ 🎯 Next Steps for You:                                  │
│                                                         │
│ 1. Advanced Drift Detection (15 min)                   │
│    You've mastered basic drift detection. Learn        │
│    advanced filtering and custom rules.                │
│    → driftmgr tutorial advanced-drift                  │
│                                                         │
│ 2. Cost Optimization (20 min)                          │
│    Discover cost-saving opportunities in your          │
│    infrastructure.                                      │
│    → driftmgr tutorial cost-optimization               │
│                                                         │
│ 3. Compliance Monitoring (25 min)                      │
│    Set up automated compliance checking for your       │
│    security frameworks.                                 │
│    → driftmgr tutorial compliance                      │
│                                                         │
│ 🏆 Skill Badges Available:                             │
│ • Multi-Cloud Expert                                   │
│ • Automation Master                                     │
│ • Security Champion                                     │
│                                                         │
│ Choose a learning path [1-3] or explore all: _         │
└─────────────────────────────────────────────────────────┘
```

## Configuration

### Help System Configuration

```yaml
# config.yaml
help_system:
  enabled: true
  
  # Context awareness
  context_detection:
    enabled: true
    collect_usage_stats: true  # For better recommendations
    error_tracking: true
  
  # Help content
  content:
    source: "embedded"  # or "remote", "local"
    update_frequency: "daily"
    cache_duration: "1h"
  
  # User interface
  ui:
    show_tooltips: true
    tooltip_delay: 500  # ms
    progressive_disclosure: true
    keyboard_shortcuts: true
  
  # Interactive features
  interactive:
    tutorials_enabled: true
    wizard_mode: true
    ai_assistant: true
    smart_suggestions: true
  
  # Troubleshooting
  troubleshooting:
    auto_diagnostics: true
    suggested_fixes: true
    problem_reporting: true
```

### Customizing Help Content

```yaml
# Custom help topics
custom_help:
  topics:
    - id: "company-aws-setup"
      title: "Company AWS Setup Guide"
      content_file: "/etc/driftmgr/help/aws-setup.md"
      context_triggers:
        - provider: "aws"
        - command: "setup"
    
    - id: "internal-compliance"
      title: "Internal Compliance Requirements"
      content_file: "/etc/driftmgr/help/compliance.md"
      context_triggers:
        - command: "compliance"
  
  # Override default content
  overrides:
    "aws-credentials": "/etc/driftmgr/help/custom-aws-creds.md"
```

## Developer Guide

### Adding New Help Content

```go
// internal/help/content.go
type HelpTopic struct {
    ID          string            `json:"id"`
    Title       string            `json:"title"`
    Content     string            `json:"content"`
    Category    string            `json:"category"`
    Tags        []string          `json:"tags"`
    Context     ContextTriggers   `json:"context"`
    Examples    []CodeExample     `json:"examples"`
    Related     []string          `json:"related"`
}

// Register new help topic
func RegisterHelpTopic(topic HelpTopic) {
    helpRegistry.Register(topic)
}

// Example usage
func init() {
    RegisterHelpTopic(HelpTopic{
        ID:       "custom-provider-setup",
        Title:    "Setting Up Custom Cloud Provider",
        Content:  loadMarkdownContent("custom-provider.md"),
        Category: "configuration",
        Tags:     []string{"setup", "provider", "custom"},
        Context: ContextTriggers{
            Commands: []string{"setup", "config"},
            Providers: []string{"custom"},
        },
        Examples: []CodeExample{
            {
                Language: "bash",
                Code:     "driftmgr config provider add --name custom --type generic",
                Description: "Add a custom provider",
            },
        },
    })
}
```

### Context Detection API

```go
// internal/help/context.go
type ContextDetector interface {
    DetectContext() HelpContext
    RegisterTrigger(trigger ContextTrigger)
}

type HelpContext struct {
    Command      string
    Provider     string
    Operation    string
    ErrorState   *ErrorContext
    UserLevel    UserExperience
    Environment  map[string]interface{}
}

// Usage in commands
func (c *DriftCommand) Execute() error {
    // ... command logic ...
    
    // Register context for help system
    helpSystem.SetContext(help.HelpContext{
        Command:   "drift",
        Operation: "detect",
        Provider:  c.Provider,
    })
    
    return nil
}
```

### Interactive Help Widgets

```go
// internal/help/widgets.go
type HelpWidget interface {
    Render() string
    HandleInput(input string) Response
}

type TutorialWidget struct {
    Steps    []TutorialStep
    Current  int
}

func (t *TutorialWidget) Render() string {
    return renderTemplate("tutorial", map[string]interface{}{
        "Step":    t.Steps[t.Current],
        "Progress": float64(t.Current) / float64(len(t.Steps)),
    })
}

// Register widget
helpSystem.RegisterWidget("tutorial", &TutorialWidget{})
```

### Testing Help Content

```go
// tests/help_test.go
func TestHelpContentAccuracy(t *testing.T) {
    tests := []struct {
        context  help.HelpContext
        expected string
    }{
        {
            context: help.HelpContext{
                Command:  "drift",
                Provider: "aws",
            },
            expected: "aws-drift-detection",
        },
    }
    
    for _, test := range tests {
        result := helpSystem.GetRelevantHelp(test.context)
        assert.Contains(t, result.TopicIDs, test.expected)
    }
}
```

## Best Practices

### Help Content Guidelines

1. **Be Contextual**: Provide help relevant to the user's current situation
2. **Progressive Disclosure**: Start with simple explanations, offer deeper detail on demand
3. **Action-Oriented**: Include concrete examples and commands users can try
4. **Error-Focused**: Anticipate common errors and provide specific solutions
5. **Up-to-Date**: Keep examples and commands current with the latest version

### User Experience Principles

1. **Don't Interrupt**: Help should be available but not intrusive
2. **Smart Defaults**: Show the most relevant help first
3. **Learn from Usage**: Improve suggestions based on user behavior
4. **Multiple Formats**: Support different learning styles (text, examples, tutorials)
5. **Accessibility**: Ensure help is accessible to users with different abilities

### Performance Considerations

1. **Lazy Loading**: Load help content only when needed
2. **Caching**: Cache frequently accessed help content
3. **Indexing**: Build search indexes for fast help lookup
4. **Offline Support**: Include essential help content in the binary

---

**Last Updated**: 2024-12-19  
**Version**: 2.0  
**Contributors**: DriftMgr Help Team

For questions about the help system, contact: help-system@driftmgr.io