# Azure Management Groups Drift Detection

This guide demonstrates enterprise-scale drift detection across Azure Management Groups and subscriptions.

## Architecture Overview

```
                 ┌─────────────────────┐
                 │   Tenant Root       │
                 │   Management Group   │
                 └──────────┬──────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
    │  Corp   │      │   IT    │      │ Sandbox │
    │   MG    │      │   MG    │      │   MG    │
    └────┬────┘      └────┬────┘      └────┬────┘
         │                │                 │
    ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
    │Production│     │   Dev   │      │  Test   │
    │   Sub    │     │   Sub   │      │  Sub    │
    └─────────┘      └─────────┘      └─────────┘
```

## Prerequisites

1. Azure CLI installed and authenticated
2. DriftMgr installed
3. Service Principal with appropriate permissions
4. Terraform state files in Azure Storage

## Step 1: Service Principal Setup

### Create Service Principal

```bash
# Create Service Principal
az ad sp create-for-rbac \
  --name "DriftMgr-ServicePrincipal" \
  --role "Reader" \
  --scopes "/providers/Microsoft.Management/managementGroups/TenantRoot"

# Output will include:
# {
#   "appId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "displayName": "DriftMgr-ServicePrincipal",
#   "password": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "tenant": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
# }
```

### Assign Permissions at Management Group Level

```bash
# Assign Reader role at root management group
az role assignment create \
  --assignee "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
  --role "Reader" \
  --scope "/providers/Microsoft.Management/managementGroups/TenantRoot"

# Assign Storage Blob Data Reader for state files
az role assignment create \
  --assignee "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
  --role "Storage Blob Data Reader" \
  --scope "/subscriptions/{subscription-id}/resourceGroups/{rg-name}/providers/Microsoft.Storage/storageAccounts/{storage-account}"
```

## Step 2: DriftMgr Configuration

### Configuration File (driftmgr-azure.yaml)

```yaml
# Azure Management Groups Configuration
version: "1.0"

provider: azure

# Authentication
auth:
  method: service_principal
  tenant_id: ${AZURE_TENANT_ID}
  client_id: ${AZURE_CLIENT_ID}
  client_secret: ${AZURE_CLIENT_SECRET}

# Management Group Hierarchy
management_groups:
  - name: "TenantRoot"
    id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    children:
      - name: "Corporate"
        id: "corp-mg-id"
        subscriptions:
          - name: "Production"
            id: "prod-sub-id"
            state_backend:
              storage_account: "tfstateprod"
              container: "terraform-state"
              key: "production.tfstate"
            resource_groups:
              - "rg-prod-networking"
              - "rg-prod-compute"
              - "rg-prod-data"
              
      - name: "IT"
        id: "it-mg-id"
        subscriptions:
          - name: "Development"
            id: "dev-sub-id"
            state_backend:
              storage_account: "tfstatedev"
              container: "terraform-state"
              key: "development.tfstate"
            resource_groups:
              - "rg-dev-all"
              
      - name: "Sandbox"
        id: "sandbox-mg-id"
        subscriptions:
          - name: "Testing"
            id: "test-sub-id"
            state_backend:
              storage_account: "tfstatetest"
              container: "terraform-state"
              key: "testing.tfstate"

# Discovery Settings
discovery:
  parallel_subscriptions: 3
  resource_types:
    include:
      - "Microsoft.Compute/virtualMachines"
      - "Microsoft.Network/virtualNetworks"
      - "Microsoft.Network/networkSecurityGroups"
      - "Microsoft.Storage/storageAccounts"
      - "Microsoft.KeyVault/vaults"
      - "Microsoft.Web/sites"
      - "Microsoft.Sql/servers"
      - "Microsoft.ContainerService/managedClusters"
    exclude:
      - "Microsoft.Insights/*"  # Exclude monitoring resources
      - "Microsoft.AlertsManagement/*"

# Drift Detection Rules
drift_rules:
  # Critical resources that should never drift
  critical_resources:
    - type: "Microsoft.Network/networkSecurityGroups"
      alert_immediately: true
    - type: "Microsoft.KeyVault/vaults"
      alert_immediately: true
    - type: "Microsoft.Authorization/roleAssignments"
      alert_immediately: true
  
  # Allowed drift for certain attributes
  allowed_drift:
    - type: "Microsoft.Compute/virtualMachines"
      attributes:
        - "tags.LastBackup"  # Backup tags can change
        - "properties.instanceView"  # Runtime state
    - type: "Microsoft.Web/sites"
      attributes:
        - "properties.state"  # App state can change

# Compliance Policies
compliance:
  policies:
    - name: "Require specific tags"
      type: "tagging"
      required_tags:
        - "Environment"
        - "Owner"
        - "CostCenter"
        - "Project"
    
    - name: "Network Security"
      type: "network"
      rules:
        - "No public IP on databases"
        - "NSG required on all subnets"
        - "No RDP/SSH from internet"
    
    - name: "Storage Security"
      type: "storage"
      rules:
        - "HTTPS only"
        - "Encryption at rest enabled"
        - "Private endpoints preferred"

# Notification Configuration
notifications:
  teams:
    webhook_url: ${TEAMS_WEBHOOK_URL}
    severity_filter: ["critical", "high"]
  
  email:
    smtp_server: "smtp.office365.com"
    smtp_port: 587
    from: "driftmgr@company.com"
    recipients:
      critical: ["security-team@company.com"]
      high: ["devops-team@company.com"]
      medium: ["dev-team@company.com"]
  
  azure_monitor:
    enabled: true
    workspace_id: ${LOG_ANALYTICS_WORKSPACE_ID}
    workspace_key: ${LOG_ANALYTICS_WORKSPACE_KEY}

# Remediation Settings
remediation:
  auto_remediate:
    enabled: false  # Manual approval required
    excluded_resource_types:
      - "Microsoft.Compute/virtualMachines"  # Never auto-remediate VMs
  
  approval_required: true
  approval_method: "teams"  # or "email", "servicenow"
```

## Step 3: PowerShell Script for Windows Environments

```powershell
# DriftMgr-Azure.ps1
# PowerShell script for Azure drift detection

param(
    [Parameter(Mandatory=$false)]
    [string]$ConfigFile = "driftmgr-azure.yaml",
    
    [Parameter(Mandatory=$false)]
    [string]$OutputPath = ".\reports",
    
    [Parameter(Mandatory=$false)]
    [string[]]$Subscriptions = @(),
    
    [Parameter(Mandatory=$false)]
    [switch]$GenerateReport,
    
    [Parameter(Mandatory=$false)]
    [switch]$SendNotifications
)

# Import required modules
Import-Module Az.Accounts
Import-Module Az.Resources

# Authenticate
function Connect-DriftMgrAzure {
    $tenantId = $env:AZURE_TENANT_ID
    $clientId = $env:AZURE_CLIENT_ID
    $clientSecret = ConvertTo-SecureString $env:AZURE_CLIENT_SECRET -AsPlainText -Force
    
    $credential = New-Object System.Management.Automation.PSCredential($clientId, $clientSecret)
    
    Connect-AzAccount -ServicePrincipal -Credential $credential -Tenant $tenantId
}

# Detect drift for a subscription
function Detect-SubscriptionDrift {
    param(
        [string]$SubscriptionId,
        [string]$SubscriptionName
    )
    
    Write-Host "Detecting drift for subscription: $SubscriptionName" -ForegroundColor Green
    
    # Set context
    Set-AzContext -SubscriptionId $SubscriptionId
    
    # Run DriftMgr
    $result = & driftmgr drift detect `
        --provider azure `
        --subscription $SubscriptionId `
        --config $ConfigFile `
        --output json
    
    # Save results
    $outputFile = Join-Path $OutputPath "drift-$SubscriptionName-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
    $result | Out-File -FilePath $outputFile -Encoding UTF8
    
    return $result | ConvertFrom-Json
}

# Main execution
function Start-DriftDetection {
    # Create output directory
    if (-not (Test-Path $OutputPath)) {
        New-Item -ItemType Directory -Path $OutputPath | Out-Null
    }
    
    # Connect to Azure
    Connect-DriftMgrAzure
    
    # Get subscriptions to check
    if ($Subscriptions.Count -eq 0) {
        $Subscriptions = Get-AzSubscription | Select-Object -ExpandProperty Id
    }
    
    $allResults = @()
    
    foreach ($subId in $Subscriptions) {
        $sub = Get-AzSubscription -SubscriptionId $subId
        $result = Detect-SubscriptionDrift -SubscriptionId $subId -SubscriptionName $sub.Name
        $allResults += $result
        
        # Check for critical drift
        if ($result.drift_summary.critical_count -gt 0) {
            Write-Warning "Critical drift detected in $($sub.Name)!"
            
            if ($SendNotifications) {
                Send-DriftNotification -Subscription $sub.Name -DriftData $result
            }
        }
    }
    
    # Generate consolidated report
    if ($GenerateReport) {
        Generate-DriftReport -Results $allResults
    }
    
    return $allResults
}

# Send notifications
function Send-DriftNotification {
    param(
        [string]$Subscription,
        [object]$DriftData
    )
    
    $webhookUrl = $env:TEAMS_WEBHOOK_URL
    
    $message = @{
        "@type" = "MessageCard"
        "@context" = "http://schema.org/extensions"
        "summary" = "Drift Detected in Azure"
        "themeColor" = "FF0000"
        "title" = "⚠️ Drift Alert: $Subscription"
        "sections" = @(
            @{
                "facts" = @(
                    @{
                        "name" = "Subscription"
                        "value" = $Subscription
                    },
                    @{
                        "name" = "Drifted Resources"
                        "value" = $DriftData.drift_summary.drifted_count
                    },
                    @{
                        "name" = "Missing Resources"
                        "value" = $DriftData.drift_summary.missing_count
                    },
                    @{
                        "name" = "Critical Issues"
                        "value" = $DriftData.drift_summary.critical_count
                    }
                )
            }
        )
        "potentialAction" = @(
            @{
                "@type" = "OpenUri"
                "name" = "View Report"
                "targets" = @(
                    @{
                        "os" = "default"
                        "uri" = "https://portal.azure.com"
                    }
                )
            }
        )
    }
    
    Invoke-RestMethod -Uri $webhookUrl -Method Post -Body ($message | ConvertTo-Json -Depth 10) -ContentType "application/json"
}

# Generate HTML report
function Generate-DriftReport {
    param(
        [array]$Results
    )
    
    Write-Host "Generating consolidated report..." -ForegroundColor Yellow
    
    & driftmgr report generate `
        --input-dir $OutputPath `
        --output-file "$OutputPath\drift-report-$(Get-Date -Format 'yyyyMMdd').html" `
        --format html `
        --include-recommendations
    
    Write-Host "Report generated successfully!" -ForegroundColor Green
}

# Execute
try {
    $results = Start-DriftDetection
    
    # Summary
    $totalDrift = ($results | Measure-Object -Property drift_summary.drifted_count -Sum).Sum
    $totalMissing = ($results | Measure-Object -Property drift_summary.missing_count -Sum).Sum
    
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "Drift Detection Complete!" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "Total Drifted Resources: $totalDrift" -ForegroundColor $(if ($totalDrift -gt 0) { "Red" } else { "Green" })
    Write-Host "Total Missing Resources: $totalMissing" -ForegroundColor $(if ($totalMissing -gt 0) { "Red" } else { "Green" })
    Write-Host "Reports saved to: $OutputPath" -ForegroundColor Yellow
    
} catch {
    Write-Error "Drift detection failed: $_"
    exit 1
}
```

## Step 4: Azure DevOps Pipeline

```yaml
# azure-pipelines.yml
trigger:
  - none  # Manual trigger or scheduled

schedules:
- cron: "0 6,18 * * *"  # Run at 6 AM and 6 PM
  displayName: Bi-daily drift detection
  branches:
    include:
    - main

pool:
  vmImage: 'ubuntu-latest'

variables:
  - group: DriftMgr-Variables  # Contains AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET

stages:
- stage: DriftDetection
  displayName: 'Detect Infrastructure Drift'
  jobs:
  - job: DetectDrift
    displayName: 'Run DriftMgr'
    steps:
    - task: AzureCLI@2
      displayName: 'Setup Azure Authentication'
      inputs:
        azureSubscription: 'DriftMgr-ServiceConnection'
        scriptType: 'bash'
        scriptLocation: 'inlineScript'
        inlineScript: |
          echo "##vso[task.setvariable variable=AZURE_TENANT_ID]$(az account show --query tenantId -o tsv)"
          echo "##vso[task.setvariable variable=AZURE_SUBSCRIPTION_ID]$(az account show --query id -o tsv)"
    
    - script: |
        # Install DriftMgr
        curl -L https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-linux-amd64 -o driftmgr
        chmod +x driftmgr
        sudo mv driftmgr /usr/local/bin/
      displayName: 'Install DriftMgr'
    
    - script: |
        # Run drift detection
        driftmgr drift detect \
          --provider azure \
          --config $(Build.SourcesDirectory)/driftmgr-azure.yaml \
          --output json \
          --save-report drift-report.json
      displayName: 'Detect Drift'
      env:
        AZURE_TENANT_ID: $(AZURE_TENANT_ID)
        AZURE_CLIENT_ID: $(AZURE_CLIENT_ID)
        AZURE_CLIENT_SECRET: $(AZURE_CLIENT_SECRET)
    
    - task: PublishBuildArtifacts@1
      displayName: 'Publish Drift Report'
      inputs:
        pathToPublish: 'drift-report.json'
        artifactName: 'drift-reports'
    
    - script: |
        # Parse results and set pipeline status
        DRIFT_COUNT=$(jq '.drift_summary.drifted_count' drift-report.json)
        CRITICAL_COUNT=$(jq '.drift_summary.critical_count' drift-report.json)
        
        if [ "$CRITICAL_COUNT" -gt 0 ]; then
          echo "##vso[task.logissue type=error]Critical drift detected! $CRITICAL_COUNT critical issues found."
          exit 1
        elif [ "$DRIFT_COUNT" -gt 0 ]; then
          echo "##vso[task.logissue type=warning]Drift detected! $DRIFT_COUNT resources have drifted."
        else
          echo "No drift detected. Infrastructure is in sync!"
        fi
      displayName: 'Analyze Results'
      condition: always()

- stage: Remediation
  displayName: 'Generate Remediation Plan'
  dependsOn: DriftDetection
  condition: and(failed(), eq(variables['Build.Reason'], 'Manual'))
  jobs:
  - job: GeneratePlan
    displayName: 'Generate Remediation'
    steps:
    - download: current
      artifact: drift-reports
    
    - script: |
        driftmgr remediate \
          --drift-report $(Pipeline.Workspace)/drift-reports/drift-report.json \
          --output-format terraform \
          --save-to remediation-plan.tf
      displayName: 'Generate Remediation Plan'
    
    - task: PublishBuildArtifacts@1
      displayName: 'Publish Remediation Plan'
      inputs:
        pathToPublish: 'remediation-plan.tf'
        artifactName: 'remediation-plans'
```

## Step 5: Azure Functions for Real-time Detection

```csharp
// DriftDetectorFunction.cs
using System;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Azure.WebJobs;
using Microsoft.Azure.WebJobs.Extensions.Http;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Logging;
using System.Diagnostics;
using Newtonsoft.Json;

public static class DriftDetectorFunction
{
    [FunctionName("DetectDrift")]
    public static async Task<IActionResult> Run(
        [HttpTrigger(AuthorizationLevel.Function, "post", Route = null)] HttpRequest req,
        ILogger log)
    {
        log.LogInformation("Drift detection triggered");

        string requestBody = await new StreamReader(req.Body).ReadToEndAsync();
        dynamic data = JsonConvert.DeserializeObject(requestBody);
        
        string subscriptionId = data?.subscriptionId;
        string resourceGroup = data?.resourceGroup;
        
        if (string.IsNullOrEmpty(subscriptionId))
        {
            return new BadRequestObjectResult("Please provide subscriptionId");
        }

        var process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = "/home/site/wwwroot/driftmgr",
                Arguments = $"drift detect --provider azure --subscription {subscriptionId} --resource-group {resourceGroup} --output json",
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            }
        };

        process.Start();
        string output = await process.StandardOutput.ReadToEndAsync();
        string error = await process.StandardError.ReadToEndAsync();
        process.WaitForExit();

        if (process.ExitCode != 0)
        {
            log.LogError($"DriftMgr failed: {error}");
            return new StatusCodeResult(500);
        }

        var result = JsonConvert.DeserializeObject(output);
        
        // Store in Cosmos DB
        await StoreDriftResults(subscriptionId, result);
        
        // Send alerts if needed
        await SendAlertsIfNeeded(result);

        return new OkObjectResult(result);
    }

    private static async Task StoreDriftResults(string subscriptionId, dynamic results)
    {
        // Store in Cosmos DB for historical tracking
        var cosmosClient = new CosmosClient(Environment.GetEnvironmentVariable("CosmosDBConnection"));
        var container = cosmosClient.GetContainer("DriftMgr", "DriftResults");
        
        var document = new
        {
            id = Guid.NewGuid().ToString(),
            subscriptionId = subscriptionId,
            timestamp = DateTime.UtcNow,
            results = results
        };
        
        await container.CreateItemAsync(document);
    }

    private static async Task SendAlertsIfNeeded(dynamic results)
    {
        int driftCount = results.drift_summary.drifted_count;
        int criticalCount = results.drift_summary.critical_count;
        
        if (criticalCount > 0)
        {
            // Send immediate alert
            await SendTeamsAlert("Critical", results);
        }
        else if (driftCount > 5)
        {
            // Send warning
            await SendTeamsAlert("Warning", results);
        }
    }

    private static async Task SendTeamsAlert(string severity, dynamic results)
    {
        // Implementation for Teams webhook notification
        var webhookUrl = Environment.GetEnvironmentVariable("TeamsWebhookUrl");
        // ... send notification
    }
}
```

## Best Practices

1. **Use Managed Identities**: Prefer Managed Identities over Service Principals when running in Azure
2. **Scope Permissions**: Grant minimal required permissions at the appropriate scope
3. **Tag Compliance**: Enforce tagging policies before drift detection
4. **Cost Management**: Use resource group filters to limit API calls
5. **Parallel Processing**: Balance between speed and API throttling
6. **State File Security**: Encrypt state files and use private endpoints

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   ```bash
   # Verify Service Principal
   az login --service-principal -u $AZURE_CLIENT_ID -p $AZURE_CLIENT_SECRET --tenant $AZURE_TENANT_ID
   az account show
   ```

2. **Permission Denied**
   ```bash
   # Check role assignments
   az role assignment list --assignee $AZURE_CLIENT_ID
   ```

3. **State File Access**
   ```bash
   # Verify storage access
   az storage blob list --account-name tfstateprod --container-name terraform-state --auth-mode login
   ```

## Integration with Azure Policy

```json
{
  "properties": {
    "displayName": "Require DriftMgr scan before deployment",
    "policyType": "Custom",
    "mode": "All",
    "description": "Ensures drift detection is run before any deployment",
    "metadata": {
      "category": "Compliance"
    },
    "policyRule": {
      "if": {
        "field": "type",
        "equals": "Microsoft.Resources/deployments"
      },
      "then": {
        "effect": "deployIfNotExists",
        "details": {
          "type": "Microsoft.Logic/workflows",
          "name": "DriftMgr-PreDeployment-Check",
          "roleDefinitionIds": [
            "/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c"
          ],
          "deployment": {
            "properties": {
              "mode": "Incremental",
              "template": {
                // Logic App template to trigger DriftMgr
              }
            }
          }
        }
      }
    }
  }
}
```

## Next Steps

- Implement automated remediation workflows
- Set up Azure Monitor dashboards
- Configure Log Analytics queries
- Create custom compliance policies
- Integrate with Azure Sentinel for security monitoring

For more information, visit the [DriftMgr documentation](https://github.com/catherinevee/driftmgr/docs).