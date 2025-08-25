# DriftMgr Cost Calculation Documentation

## Overview

DriftMgr provides intelligent cost analysis for infrastructure drift, helping teams understand the financial impact of both detected drift and proposed remediation actions. This document explains how DriftMgr calculates costs associated with drift remediation.

## Cost Calculation Components

### 1. Resource Pricing Data

DriftMgr maintains pricing databases for each cloud provider:

- **AWS**: EC2, RDS, S3, EBS, Load Balancers, and 50+ other services
- **Azure**: VMs, Storage, SQL Database, App Services, and 40+ other services  
- **GCP**: Compute Engine, Cloud Storage, Cloud SQL, and 30+ other services
- **DigitalOcean**: Droplets, Volumes, Databases, and Spaces

Pricing data is updated daily from official provider APIs and includes:
- Regional pricing variations
- Reserved instance discounts
- Spot instance pricing
- Volume-based discounts
- Data transfer costs

### 2. Cost Impact Categories

DriftMgr categorizes cost impacts into several types:

#### Immediate Costs
Direct costs that will be incurred immediately upon remediation:
- Instance resizing (up or down)
- Storage modifications
- Network configuration changes
- License changes

#### Ongoing Costs
Recurring monthly or annual costs:
- Compute instance hours
- Storage capacity
- Data transfer rates
- Backup retention
- Reserved capacity

#### One-Time Costs
Non-recurring costs associated with remediation:
- Snapshot creation
- Data migration
- Resource recreation
- Temporary duplicate resources during migration

#### Potential Savings
Cost reductions from remediation:
- Downsizing oversized resources
- Removing unused resources
- Optimizing storage tiers
- Consolidating redundant resources

## Cost Calculation Methods

### 1. Instance Resizing Calculations

```
Monthly Cost Delta = (New Instance Price - Current Instance Price) * Hours per Month * Quantity

Example:
Current: t2.micro ($0.0116/hour)
Drift: t2.small ($0.023/hour)
Delta: ($0.023 - $0.0116) * 730 hours = +$8.32/month
```

### 2. Storage Cost Calculations

```
Storage Cost = Volume Size (GB) * Price per GB/month * Number of Volumes

Additional costs:
- IOPS charges for provisioned IOPS volumes
- Snapshot storage costs
- Cross-region replication costs
```

### 3. Data Transfer Calculations

```
Transfer Cost = Data Volume (GB) * Transfer Rate

Considers:
- Intra-region transfers (usually free)
- Inter-region transfers
- Internet egress charges
- VPN/Direct Connect fees
```

### 4. Backup and Retention Costs

```
Backup Cost = Storage Size * Backup Storage Rate * Retention Days / 30

Example:
100GB database * $0.05/GB/month * 30 days retention = $5.00/month
```

## Real-World Cost Calculation Examples

### Example 1: EC2 Instance Type Change

**Drift Detected**: Instance changed from t2.micro to t2.small

```
Current State (t2.micro):
- Hourly: $0.0116
- Monthly: $8.47
- Annual: $101.64

Actual State (t2.small):
- Hourly: $0.023
- Monthly: $16.79
- Annual: $201.48

Cost Impact:
- Monthly increase: +$8.32
- Annual increase: +$99.84
- Remediation action: Resize back to t2.micro
- Savings from remediation: $8.32/month
```

### Example 2: RDS Backup Retention Change

**Drift Detected**: Backup retention reduced from 30 to 7 days

```
Database Size: 500GB
Backup Storage Rate: $0.095/GB/month

Original (30 days):
- Daily backup size: 500GB
- Storage required: 500GB * 30 = 15,000GB
- Monthly cost: 15,000GB * $0.095 = $1,425

Current (7 days):
- Storage required: 500GB * 7 = 3,500GB
- Monthly cost: 3,500GB * $0.095 = $332.50

Cost Impact:
- Monthly savings: -$1,092.50
- Compliance risk: HIGH (violates retention policy)
- Remediation cost: +$1,092.50/month to restore compliance
```

### Example 3: Unencrypted S3 Bucket

**Drift Detected**: S3 bucket encryption disabled

```
Bucket Size: 10TB
Current: No encryption (standard storage)
Required: AES-256 encryption

Cost Analysis:
- Encryption overhead: ~0% (AWS S3 encryption is free)
- Performance impact: Negligible
- Compliance value: Critical for SOC2/HIPAA

Remediation Cost: $0 (no additional charges for S3 encryption)
Risk Mitigation Value: HIGH
```

### Example 4: Multi-Resource Drift Scenario

**Drift Detected**: Web application infrastructure changes

```
Components:
1. ALB: Additional availability zone (+$16.20/month)
2. EC2: 3 instances upsized (+$24.96/month)
3. RDS: Multi-AZ disabled (-$127.00/month)
4. S3: Lifecycle policy removed (+$45.00/month in retained objects)

Total Cost Impact:
- Immediate: -$40.84/month
- Risk: HIGH (no Multi-AZ for database)
- Recommended action: Re-enable Multi-AZ despite cost
```

## Cost Optimization Intelligence

### Smart Recommendations

DriftMgr provides intelligent recommendations based on:

1. **Usage Patterns**
   - Identifies underutilized resources
   - Suggests rightsizing opportunities
   - Recommends reserved instance purchases

2. **Cost-Risk Analysis**
   - Balances cost savings against operational risk
   - Prioritizes security and compliance over cost
   - Provides TCO (Total Cost of Ownership) analysis

3. **Remediation Strategies**
   - Batch similar changes to minimize downtime
   - Schedule during maintenance windows
   - Use gradual rollout for large-scale changes

### Cost Calculation Factors

DriftMgr considers multiple factors when calculating costs:

```yaml
cost_factors:
  compute:
    - instance_type
    - operating_system
    - region
    - tenancy (shared/dedicated)
    - pricing_model (on-demand/reserved/spot)
    
  storage:
    - volume_type (gp2/gp3/io1/io2)
    - size_gb
    - iops_provisioned
    - throughput_mbps
    - snapshot_frequency
    
  network:
    - data_transfer_gb
    - elastic_ips
    - nat_gateways
    - load_balancers
    - vpn_connections
    
  database:
    - engine_type
    - instance_class
    - storage_size
    - multi_az
    - read_replicas
    - backup_retention
    
  auxiliary:
    - monitoring_detailed
    - log_retention
    - support_tier
```

## API Integration

### Cost Calculation API Endpoint

```bash
GET /api/v1/drift/cost-analysis

Response:
{
  "drift_id": "drift-123456",
  "total_impact": {
    "monthly": 156.78,
    "annual": 1881.36,
    "currency": "USD"
  },
  "breakdown": [
    {
      "resource": "i-0abc123",
      "type": "ec2_instance",
      "change": "instance_type",
      "from": "t2.micro",
      "to": "t2.small",
      "cost_delta": {
        "monthly": 8.32,
        "annual": 99.84
      }
    }
  ],
  "recommendations": [
    {
      "action": "resize_instance",
      "savings": {
        "monthly": 8.32,
        "annual": 99.84
      },
      "risk": "low",
      "downtime": "2 minutes"
    }
  ]
}
```

### Bulk Cost Analysis

```bash
POST /api/v1/drift/bulk-cost-analysis

Request:
{
  "resource_ids": ["i-0abc123", "db-prod", "s3-bucket-1"],
  "include_recommendations": true,
  "currency": "USD"
}
```

## Configuration

### Setting Cost Preferences

```yaml
# configs/cost-settings.yaml
cost_analysis:
  currency: USD
  include_tax: false
  discount_rate: 0.10  # 10% enterprise discount
  
  thresholds:
    notify_increase: 100    # Alert if costs increase by $100/month
    auto_remediate: 50      # Auto-fix if savings > $50/month
    
  pricing_source:
    aws: "api"              # Use real-time API pricing
    azure: "cache"          # Use cached pricing data
    gcp: "manual"           # Use manually configured rates
    
  factors:
    include_data_transfer: true
    include_support_costs: true
    include_tax_estimates: false
    amortize_reserved: true
```

### Custom Pricing Overrides

For private pricing agreements or enterprise discounts:

```yaml
# configs/custom-pricing.yaml
custom_pricing:
  aws:
    ec2:
      t2.micro: 0.0104    # 10% discount
      t2.small: 0.0207    # 10% discount
    rds:
      discount_percentage: 15
      
  azure:
    vms:
      enterprise_agreement: true
      discount_percentage: 20
```

## Cost Reports

### Monthly Cost Impact Report

```
DRIFT COST IMPACT REPORT - January 2024
============================================================

SUMMARY
-------
Total Monthly Impact: +$457.32
Total Annual Impact: +$5,487.84
Resources Affected: 23

BREAKDOWN BY SERVICE
--------------------
EC2:        +$234.56 (12 instances)
RDS:        +$156.78 (3 databases)
S3:         +$45.99 (5 buckets)
ALB:        +$20.00 (2 load balancers)

TOP COST INCREASES
-------------------
1. i-0abc123def: +$87.43/month (m5.large -> m5.xlarge)
2. db-prod-01: +$76.54/month (backup retention 7 -> 30 days)
3. s3-logs-bucket: +$45.99/month (lifecycle policy removed)

OPTIMIZATION OPPORTUNITIES
--------------------------
1. Resize 3 over-provisioned instances: Save $156/month
2. Delete 2 unused elastic IPs: Save $7.20/month
3. Optimize S3 storage classes: Save $34/month

RECOMMENDED ACTIONS
-------------------
Priority | Resource | Action | Monthly Savings | Risk
---------|----------|--------|-----------------|------
HIGH     | i-0def456 | Resize to t2.micro | $8.32 | Low
HIGH     | db-staging | Disable Multi-AZ | $127.00 | Medium
MEDIUM   | s3-archive | Enable lifecycle | $45.99 | Low
```

## Best Practices

### 1. Regular Cost Reviews
- Schedule weekly cost impact assessments
- Set up automated cost alerts
- Review remediation costs before execution

### 2. Cost-Aware Remediation
- Batch low-impact changes
- Schedule high-cost changes during budget cycles
- Consider gradual remediation for major changes

### 3. Budget Integration
- Set monthly/quarterly budgets
- Configure auto-remediation limits
- Track cost trends over time

### 4. Compliance vs Cost
- Never compromise security for cost savings
- Document compliance-required expenses
- Maintain audit trail of cost decisions

## Limitations

### Current Limitations
- Pricing data may lag 24 hours behind provider changes
- Complex pricing models (e.g., AWS Savings Plans) require manual configuration
- Data transfer costs are estimates based on historical patterns
- Some regional pricing may not be available

### Planned Enhancements
- Real-time pricing API integration
- Machine learning for cost prediction
- Multi-currency support
- Integration with cloud provider cost management tools

## Troubleshooting

### Common Issues

1. **Incorrect cost calculations**
   - Verify pricing data is up-to-date
   - Check custom pricing overrides
   - Ensure correct region is selected

2. **Missing cost data**
   - Enable cost analysis in configuration
   - Verify API credentials have billing access
   - Check network connectivity to pricing APIs

3. **Unexpected high costs**
   - Review resource tags and metadata
   - Verify instance types and sizes
   - Check for hidden costs (data transfer, snapshots)

## API References

- [AWS Pricing API](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/price-changes.html)
- [Azure Pricing API](https://docs.microsoft.com/en-us/rest/api/cost-management/)
- [GCP Pricing API](https://cloud.google.com/billing/docs/reference/rest)
- [DigitalOcean Pricing](https://docs.digitalocean.com/reference/api/api-reference/#tag/Billing)

## Conclusion

DriftMgr's cost calculation engine provides comprehensive financial analysis of infrastructure drift, enabling teams to make informed decisions about remediation priorities. By understanding both the immediate and long-term cost implications of drift, organizations can maintain compliance and security while optimizing cloud spending.