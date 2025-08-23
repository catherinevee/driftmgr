#!/bin/bash

# Comprehensive Reconciliation Report Generator
# Compares DriftMgr findings with multiple verification sources

echo "=== DriftMgr Reconciliation Report Generator ==="
echo ""

# Configuration
REPORT_DIR="reconciliation_reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/reconciliation_report_$TIMESTAMP.html"
JSON_REPORT="$REPORT_DIR/reconciliation_data_$TIMESTAMP.json"

# Create report directory
mkdir -p $REPORT_DIR

# Function to collect DriftMgr data
collect_driftmgr_data() {
    echo "Collecting DriftMgr discovery data..."
    
    # Run DriftMgr discovery for all providers
    ./driftmgr.exe discover --auto --format json --output $REPORT_DIR/driftmgr_discovery.json
    
    # Extract counts
    DRIFT_AWS=$(jq '.aws.resource_count // 0' $REPORT_DIR/driftmgr_discovery.json)
    DRIFT_AZURE=$(jq '.azure.resource_count // 0' $REPORT_DIR/driftmgr_discovery.json)
    DRIFT_GCP=$(jq '.gcp.resource_count // 0' $REPORT_DIR/driftmgr_discovery.json)
    DRIFT_DO=$(jq '.digitalocean.resource_count // 0' $REPORT_DIR/driftmgr_discovery.json)
    
    echo "✓ DriftMgr data collected"
}

# Function to collect AWS data
collect_aws_data() {
    echo "Collecting AWS verification data..."
    
    # AWS CLI counts
    AWS_EC2=$(aws ec2 describe-instances --query 'Reservations[*].Instances[*].[InstanceId]' --output text 2>/dev/null | grep -c '^i-' || echo 0)
    AWS_S3=$(aws s3api list-buckets --query 'Buckets[*].Name' --output text 2>/dev/null | wc -w || echo 0)
    AWS_RDS=$(aws rds describe-db-instances --query 'DBInstances[*].DBInstanceIdentifier' --output text 2>/dev/null | wc -w || echo 0)
    AWS_LAMBDA=$(aws lambda list-functions --query 'Functions[*].FunctionName' --output text 2>/dev/null | wc -w || echo 0)
    AWS_VPC=$(aws ec2 describe-vpcs --query 'Vpcs[*].VpcId' --output text 2>/dev/null | wc -w || echo 0)
    
    AWS_CLI_TOTAL=$((AWS_EC2 + AWS_S3 + AWS_RDS + AWS_LAMBDA + AWS_VPC))
    
    # AWS Config count (if available)
    AWS_CONFIG_COUNT=$(aws configservice select-resource-config \
        --expression "SELECT COUNT(*) WHERE resourceType LIKE 'AWS::%'" \
        --output text 2>/dev/null | grep -o '[0-9]*' | head -1 || echo "N/A")
    
    # AWS Cost Explorer resource count (approximation)
    AWS_COST_COUNT=$(aws ce get-cost-and-usage \
        --time-period Start=$(date -d '1 day ago' +%Y-%m-%d),End=$(date +%Y-%m-%d) \
        --granularity DAILY \
        --metrics UsageQuantity \
        --group-by Type=DIMENSION,Key=SERVICE \
        --query 'ResultsByTime[0].Groups | length(@)' \
        --output text 2>/dev/null || echo "N/A")
    
    echo "✓ AWS data collected"
}

# Function to collect Azure data
collect_azure_data() {
    echo "Collecting Azure verification data..."
    
    # Azure CLI count
    AZURE_CLI_TOTAL=$(az resource list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    
    # Azure Resource Graph count
    AZURE_GRAPH_COUNT=$(az graph query -q "Resources | summarize count()" --query 'data[0].count_' --output tsv 2>/dev/null || echo "N/A")
    
    # Azure by resource type
    az resource list --query "groupBy(@, &type)[].{type: @[0].type, count: length(@)}" --output json > $REPORT_DIR/azure_types.json 2>/dev/null
    
    echo "✓ Azure data collected"
}

# Function to collect GCP data
collect_gcp_data() {
    echo "Collecting GCP verification data..."
    
    PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
    
    # GCP resource counts
    GCP_INSTANCES=$(gcloud compute instances list --format='value(name)' 2>/dev/null | wc -l || echo 0)
    GCP_BUCKETS=$(gcloud storage buckets list --format='value(name)' 2>/dev/null | wc -l || echo 0)
    GCP_SQL=$(gcloud sql instances list --format='value(name)' 2>/dev/null | wc -l || echo 0)
    GCP_FUNCTIONS=$(gcloud functions list --format='value(name)' 2>/dev/null | wc -l || echo 0)
    
    GCP_CLI_TOTAL=$((GCP_INSTANCES + GCP_BUCKETS + GCP_SQL + GCP_FUNCTIONS))
    
    # GCP Asset Inventory count
    GCP_ASSET_COUNT=$(gcloud asset search-all-resources --project=$PROJECT_ID --format='value(name)' 2>/dev/null | wc -l || echo "N/A")
    
    echo "✓ GCP data collected"
}

# Function to collect DigitalOcean data
collect_do_data() {
    echo "Collecting DigitalOcean verification data..."
    
    # DigitalOcean counts
    DO_DROPLETS=$(doctl compute droplet list --format ID --no-header 2>/dev/null | wc -l || echo 0)
    DO_VOLUMES=$(doctl compute volume list --format ID --no-header 2>/dev/null | wc -l || echo 0)
    DO_DBS=$(doctl databases list --format ID --no-header 2>/dev/null | wc -l || echo 0)
    DO_LBS=$(doctl compute load-balancer list --format ID --no-header 2>/dev/null | wc -l || echo 0)
    
    DO_CLI_TOTAL=$((DO_DROPLETS + DO_VOLUMES + DO_DBS + DO_LBS))
    
    echo "✓ DigitalOcean data collected"
}

# Function to generate JSON data
generate_json_data() {
    cat > $JSON_REPORT <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "providers": {
    "aws": {
      "driftmgr_count": $DRIFT_AWS,
      "cli_count": $AWS_CLI_TOTAL,
      "config_count": "$AWS_CONFIG_COUNT",
      "cost_explorer_services": "$AWS_COST_COUNT",
      "details": {
        "ec2": $AWS_EC2,
        "s3": $AWS_S3,
        "rds": $AWS_RDS,
        "lambda": $AWS_LAMBDA,
        "vpc": $AWS_VPC
      },
      "variance": $((DRIFT_AWS - AWS_CLI_TOTAL)),
      "variance_percentage": $(echo "scale=2; ($DRIFT_AWS - $AWS_CLI_TOTAL) * 100 / $AWS_CLI_TOTAL" | bc 2>/dev/null || echo 0)
    },
    "azure": {
      "driftmgr_count": $DRIFT_AZURE,
      "cli_count": $AZURE_CLI_TOTAL,
      "graph_count": "$AZURE_GRAPH_COUNT",
      "variance": $((DRIFT_AZURE - AZURE_CLI_TOTAL)),
      "variance_percentage": $(echo "scale=2; ($DRIFT_AZURE - $AZURE_CLI_TOTAL) * 100 / $AZURE_CLI_TOTAL" | bc 2>/dev/null || echo 0)
    },
    "gcp": {
      "driftmgr_count": $DRIFT_GCP,
      "cli_count": $GCP_CLI_TOTAL,
      "asset_inventory_count": "$GCP_ASSET_COUNT",
      "details": {
        "instances": $GCP_INSTANCES,
        "buckets": $GCP_BUCKETS,
        "sql": $GCP_SQL,
        "functions": $GCP_FUNCTIONS
      },
      "variance": $((DRIFT_GCP - GCP_CLI_TOTAL)),
      "variance_percentage": $(echo "scale=2; ($DRIFT_GCP - $GCP_CLI_TOTAL) * 100 / $GCP_CLI_TOTAL" | bc 2>/dev/null || echo 0)
    },
    "digitalocean": {
      "driftmgr_count": $DRIFT_DO,
      "cli_count": $DO_CLI_TOTAL,
      "details": {
        "droplets": $DO_DROPLETS,
        "volumes": $DO_VOLUMES,
        "databases": $DO_DBS,
        "load_balancers": $DO_LBS
      },
      "variance": $((DRIFT_DO - DO_CLI_TOTAL)),
      "variance_percentage": $(echo "scale=2; ($DRIFT_DO - $DO_CLI_TOTAL) * 100 / $DO_CLI_TOTAL" | bc 2>/dev/null || echo 0)
    }
  },
  "summary": {
    "total_driftmgr": $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO)),
    "total_cli": $((AWS_CLI_TOTAL + AZURE_CLI_TOTAL + GCP_CLI_TOTAL + DO_CLI_TOTAL)),
    "total_variance": $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO - AWS_CLI_TOTAL - AZURE_CLI_TOTAL - GCP_CLI_TOTAL - DO_CLI_TOTAL)),
    "verification_status": "$([ $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO)) -eq $((AWS_CLI_TOTAL + AZURE_CLI_TOTAL + GCP_CLI_TOTAL + DO_CLI_TOTAL)) ] && echo 'PASSED' || echo 'VARIANCE_DETECTED')"
  }
}
EOF
}

# Function to generate HTML report
generate_html_report() {
    cat > $REPORT_FILE <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DriftMgr Reconciliation Report - $(date +%Y-%m-%d)</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
        }
        h1 {
            margin: 0;
            font-size: 2.5em;
        }
        .timestamp {
            opacity: 0.9;
            margin-top: 10px;
        }
        .summary-cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .card h3 {
            margin-top: 0;
            color: #667eea;
        }
        .metric {
            font-size: 2em;
            font-weight: bold;
            color: #333;
        }
        .status-badge {
            display: inline-block;
            padding: 5px 10px;
            border-radius: 20px;
            font-size: 0.9em;
            font-weight: bold;
        }
        .status-passed {
            background: #d4edda;
            color: #155724;
        }
        .status-variance {
            background: #fff3cd;
            color: #856404;
        }
        .status-failed {
            background: #f8d7da;
            color: #721c24;
        }
        table {
            width: 100%;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        th {
            background: #667eea;
            color: white;
            padding: 15px;
            text-align: left;
        }
        td {
            padding: 15px;
            border-bottom: 1px solid #e0e0e0;
        }
        tr:last-child td {
            border-bottom: none;
        }
        tr:hover {
            background: #f8f9fa;
        }
        .variance-positive {
            color: #dc3545;
        }
        .variance-negative {
            color: #ffc107;
        }
        .variance-zero {
            color: #28a745;
        }
        .chart-container {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .progress-bar {
            width: 100%;
            height: 30px;
            background: #e0e0e0;
            border-radius: 15px;
            overflow: hidden;
            margin: 10px 0;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea, #764ba2);
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: bold;
        }
        .details-section {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .recommendations {
            background: #e8f4fd;
            border-left: 4px solid #667eea;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
        }
        .footer {
            text-align: center;
            color: #666;
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #e0e0e0;
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div class="header">
        <h1>DriftMgr Reconciliation Report</h1>
        <div class="timestamp">Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')</div>
    </div>

    <div class="summary-cards">
        <div class="card">
            <h3>Total Resources (DriftMgr)</h3>
            <div class="metric">$((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO))</div>
        </div>
        <div class="card">
            <h3>Total Resources (CLI Tools)</h3>
            <div class="metric">$((AWS_CLI_TOTAL + AZURE_CLI_TOTAL + GCP_CLI_TOTAL + DO_CLI_TOTAL))</div>
        </div>
        <div class="card">
            <h3>Total Variance</h3>
            <div class="metric $([ $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO - AWS_CLI_TOTAL - AZURE_CLI_TOTAL - GCP_CLI_TOTAL - DO_CLI_TOTAL)) -eq 0 ] && echo 'variance-zero' || echo 'variance-positive')">
                $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO - AWS_CLI_TOTAL - AZURE_CLI_TOTAL - GCP_CLI_TOTAL - DO_CLI_TOTAL))
            </div>
        </div>
        <div class="card">
            <h3>Verification Status</h3>
            <span class="status-badge $([ $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO)) -eq $((AWS_CLI_TOTAL + AZURE_CLI_TOTAL + GCP_CLI_TOTAL + DO_CLI_TOTAL)) ] && echo 'status-passed' || echo 'status-variance')">
                $([ $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO)) -eq $((AWS_CLI_TOTAL + AZURE_CLI_TOTAL + GCP_CLI_TOTAL + DO_CLI_TOTAL)) ] && echo 'PASSED' || echo 'VARIANCE DETECTED')
            </span>
        </div>
    </div>

    <h2>Provider Comparison</h2>
    <table>
        <thead>
            <tr>
                <th>Provider</th>
                <th>DriftMgr Count</th>
                <th>CLI Tool Count</th>
                <th>Config/Inventory Service</th>
                <th>Variance</th>
                <th>Status</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td><strong>AWS</strong></td>
                <td>$DRIFT_AWS</td>
                <td>$AWS_CLI_TOTAL</td>
                <td>$AWS_CONFIG_COUNT</td>
                <td class="$([ $((DRIFT_AWS - AWS_CLI_TOTAL)) -eq 0 ] && echo 'variance-zero' || echo 'variance-positive')">
                    $((DRIFT_AWS - AWS_CLI_TOTAL))
                </td>
                <td>
                    <span class="status-badge $([ $((DRIFT_AWS - AWS_CLI_TOTAL)) -eq 0 ] && echo 'status-passed' || echo 'status-variance')">
                        $([ $((DRIFT_AWS - AWS_CLI_TOTAL)) -eq 0 ] && echo '✓' || echo '⚠')
                    </span>
                </td>
            </tr>
            <tr>
                <td><strong>Azure</strong></td>
                <td>$DRIFT_AZURE</td>
                <td>$AZURE_CLI_TOTAL</td>
                <td>$AZURE_GRAPH_COUNT</td>
                <td class="$([ $((DRIFT_AZURE - AZURE_CLI_TOTAL)) -eq 0 ] && echo 'variance-zero' || echo 'variance-positive')">
                    $((DRIFT_AZURE - AZURE_CLI_TOTAL))
                </td>
                <td>
                    <span class="status-badge $([ $((DRIFT_AZURE - AZURE_CLI_TOTAL)) -eq 0 ] && echo 'status-passed' || echo 'status-variance')">
                        $([ $((DRIFT_AZURE - AZURE_CLI_TOTAL)) -eq 0 ] && echo '✓' || echo '⚠')
                    </span>
                </td>
            </tr>
            <tr>
                <td><strong>GCP</strong></td>
                <td>$DRIFT_GCP</td>
                <td>$GCP_CLI_TOTAL</td>
                <td>$GCP_ASSET_COUNT</td>
                <td class="$([ $((DRIFT_GCP - GCP_CLI_TOTAL)) -eq 0 ] && echo 'variance-zero' || echo 'variance-positive')">
                    $((DRIFT_GCP - GCP_CLI_TOTAL))
                </td>
                <td>
                    <span class="status-badge $([ $((DRIFT_GCP - GCP_CLI_TOTAL)) -eq 0 ] && echo 'status-passed' || echo 'status-variance')">
                        $([ $((DRIFT_GCP - GCP_CLI_TOTAL)) -eq 0 ] && echo '✓' || echo '⚠')
                    </span>
                </td>
            </tr>
            <tr>
                <td><strong>DigitalOcean</strong></td>
                <td>$DRIFT_DO</td>
                <td>$DO_CLI_TOTAL</td>
                <td>N/A</td>
                <td class="$([ $((DRIFT_DO - DO_CLI_TOTAL)) -eq 0 ] && echo 'variance-zero' || echo 'variance-positive')">
                    $((DRIFT_DO - DO_CLI_TOTAL))
                </td>
                <td>
                    <span class="status-badge $([ $((DRIFT_DO - DO_CLI_TOTAL)) -eq 0 ] && echo 'status-passed' || echo 'status-variance')">
                        $([ $((DRIFT_DO - DO_CLI_TOTAL)) -eq 0 ] && echo '✓' || echo '⚠')
                    </span>
                </td>
            </tr>
        </tbody>
    </table>

    <div class="chart-container">
        <h2>Resource Distribution</h2>
        <canvas id="resourceChart" width="400" height="100"></canvas>
    </div>

    <div class="details-section">
        <h2>AWS Resource Breakdown</h2>
        <table>
            <tr><td>EC2 Instances</td><td>$AWS_EC2</td></tr>
            <tr><td>S3 Buckets</td><td>$AWS_S3</td></tr>
            <tr><td>RDS Instances</td><td>$AWS_RDS</td></tr>
            <tr><td>Lambda Functions</td><td>$AWS_LAMBDA</td></tr>
            <tr><td>VPCs</td><td>$AWS_VPC</td></tr>
        </table>
    </div>

    <div class="details-section">
        <h2>GCP Resource Breakdown</h2>
        <table>
            <tr><td>Compute Instances</td><td>$GCP_INSTANCES</td></tr>
            <tr><td>Storage Buckets</td><td>$GCP_BUCKETS</td></tr>
            <tr><td>Cloud SQL</td><td>$GCP_SQL</td></tr>
            <tr><td>Cloud Functions</td><td>$GCP_FUNCTIONS</td></tr>
        </table>
    </div>

    <div class="details-section">
        <h2>DigitalOcean Resource Breakdown</h2>
        <table>
            <tr><td>Droplets</td><td>$DO_DROPLETS</td></tr>
            <tr><td>Volumes</td><td>$DO_VOLUMES</td></tr>
            <tr><td>Databases</td><td>$DO_DBS</td></tr>
            <tr><td>Load Balancers</td><td>$DO_LBS</td></tr>
        </table>
    </div>

    <div class="recommendations">
        <h2>📋 Recommendations</h2>
        <ul>
            $([ $((DRIFT_AWS - AWS_CLI_TOTAL)) -ne 0 ] && echo "<li>AWS shows a variance of $((DRIFT_AWS - AWS_CLI_TOTAL)) resources. Review AWS Config for detailed inventory.</li>")
            $([ $((DRIFT_AZURE - AZURE_CLI_TOTAL)) -ne 0 ] && echo "<li>Azure shows a variance of $((DRIFT_AZURE - AZURE_CLI_TOTAL)) resources. Check Azure Resource Graph for discrepancies.</li>")
            $([ $((DRIFT_GCP - GCP_CLI_TOTAL)) -ne 0 ] && echo "<li>GCP shows a variance of $((DRIFT_GCP - GCP_CLI_TOTAL)) resources. Verify with Cloud Asset Inventory.</li>")
            $([ $((DRIFT_DO - DO_CLI_TOTAL)) -ne 0 ] && echo "<li>DigitalOcean shows a variance of $((DRIFT_DO - DO_CLI_TOTAL)) resources. Confirm with doctl inventory.</li>")
            <li>Consider implementing automated tagging for better resource tracking</li>
            <li>Review unmanaged resources for potential security risks</li>
            <li>Update discovery configurations to include all resource types</li>
        </ul>
    </div>

    <div class="footer">
        <p>Generated by DriftMgr Reconciliation System</p>
        <p>Report ID: $TIMESTAMP | <a href="$JSON_REPORT">Download JSON Data</a></p>
    </div>

    <script>
        // Create comparison chart
        const ctx = document.getElementById('resourceChart').getContext('2d');
        const resourceChart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['AWS', 'Azure', 'GCP', 'DigitalOcean'],
                datasets: [{
                    label: 'DriftMgr',
                    data: [$DRIFT_AWS, $DRIFT_AZURE, $DRIFT_GCP, $DRIFT_DO],
                    backgroundColor: 'rgba(102, 126, 234, 0.8)',
                    borderColor: 'rgba(102, 126, 234, 1)',
                    borderWidth: 1
                }, {
                    label: 'CLI Tools',
                    data: [$AWS_CLI_TOTAL, $AZURE_CLI_TOTAL, $GCP_CLI_TOTAL, $DO_CLI_TOTAL],
                    backgroundColor: 'rgba(118, 75, 162, 0.8)',
                    borderColor: 'rgba(118, 75, 162, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    </script>
</body>
</html>
EOF
}

# Main execution
echo "Starting reconciliation report generation..."
echo ""

# Collect all data
collect_driftmgr_data
collect_aws_data
collect_azure_data
collect_gcp_data
collect_do_data

# Generate reports
generate_json_data
generate_html_report

echo ""
echo "=== Reconciliation Report Generated ==="
echo ""
echo "Reports created:"
echo "  📄 HTML Report: $REPORT_FILE"
echo "  📊 JSON Data: $JSON_REPORT"
echo ""
echo "Summary:"
echo "  Total DriftMgr: $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO)) resources"
echo "  Total CLI Tools: $((AWS_CLI_TOTAL + AZURE_CLI_TOTAL + GCP_CLI_TOTAL + DO_CLI_TOTAL)) resources"
echo "  Total Variance: $((DRIFT_AWS + DRIFT_AZURE + DRIFT_GCP + DRIFT_DO - AWS_CLI_TOTAL - AZURE_CLI_TOTAL - GCP_CLI_TOTAL - DO_CLI_TOTAL)) resources"
echo ""

# Open report in browser if available
if command -v xdg-open &> /dev/null; then
    xdg-open $REPORT_FILE
elif command -v open &> /dev/null; then
    open $REPORT_FILE
elif command -v start &> /dev/null; then
    start $REPORT_FILE
else
    echo "Open $REPORT_FILE in your browser to view the report"
fi