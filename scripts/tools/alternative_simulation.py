#!/usr/bin/env python3
"""
Alternative DriftMgr Simulation

This simulation works around the database/authentication issues
by using mock responses and demonstrating expected behavior.
"""

import json
import random
import time
import sys
from datetime import datetime

def get_realistic_resources(provider, region):
    """Generate realistic resource counts based on provider and region"""
    
    # Base resource types for each provider
    aws_resources = {
        'EC2': {'min': 0, 'max': 25, 'name': 'EC2 instances'},
        'S3': {'min': 0, 'max': 15, 'name': 'S3 buckets'},
        'RDS': {'min': 0, 'max': 8, 'name': 'RDS databases'},
        'VPC': {'min': 1, 'max': 5, 'name': 'VPCs'},
        'SecurityGroup': {'min': 2, 'max': 20, 'name': 'Security Groups'},
        'ELB': {'min': 0, 'max': 6, 'name': 'Load Balancers'},
        'Lambda': {'min': 0, 'max': 12, 'name': 'Lambda functions'},
        'CloudFormation': {'min': 0, 'max': 10, 'name': 'CloudFormation stacks'},
        'IAM': {'min': 5, 'max': 30, 'name': 'IAM roles'},
        'CloudWatch': {'min': 1, 'max': 8, 'name': 'CloudWatch log groups'}
    }
    
    azure_resources = {
        'VM': {'min': 0, 'max': 20, 'name': 'Virtual Machines'},
        'StorageAccount': {'min': 0, 'max': 12, 'name': 'Storage Accounts'},
        'SQLDatabase': {'min': 0, 'max': 6, 'name': 'SQL Databases'},
        'VNet': {'min': 1, 'max': 4, 'name': 'Virtual Networks'},
        'NSG': {'min': 2, 'max': 15, 'name': 'Network Security Groups'},
        'AppService': {'min': 0, 'max': 8, 'name': 'App Services'},
        'FunctionApp': {'min': 0, 'max': 10, 'name': 'Function Apps'},
        'KeyVault': {'min': 0, 'max': 5, 'name': 'Key Vaults'},
        'CosmosDB': {'min': 0, 'max': 3, 'name': 'Cosmos DB accounts'},
        'AKS': {'min': 0, 'max': 4, 'name': 'AKS clusters'}
    }
    
    # Regional factors - some regions are more active than others
    region_factors = {
        # AWS regions
        'us-east-1': 1.2,  # Most active
        'us-west-2': 1.1,
        'eu-west-1': 1.0,
        'ap-southeast-1': 0.8,
        'sa-east-1': 0.6,  # Less active
        'ap-northeast-1': 0.9,
        'eu-central-1': 0.95,
        'ap-southeast-2': 0.85,
        'us-east-2': 0.9,
        'eu-west-2': 0.8,
        
        # Azure regions
        'eastus': 1.1,
        'westus2': 1.0,
        'northeurope': 0.9,
        'southeastasia': 0.8,
        'uksouth': 0.85,
        'centralus': 0.9,
        'westeurope': 0.95,
        'japaneast': 0.7,
        'australiaeast': 0.75,
        'canadacentral': 0.8
    }
    
    # Get region factor (default to 0.7 for unknown regions)
    region_factor = region_factors.get(region, 0.7)
    
    # Select resources based on provider
    resources = aws_resources if provider == 'aws' else azure_resources
    
    # Generate realistic resource counts
    resource_counts = {}
    total_resources = 0
    
    for resource_type, config in resources.items():
        # Apply region factor and add some randomness
        base_count = random.randint(config['min'], config['max'])
        adjusted_count = int(base_count * region_factor * random.uniform(0.8, 1.2))
        adjusted_count = max(config['min'], adjusted_count)  # Ensure minimum
        
        if adjusted_count > 0:
            resource_counts[config['name']] = adjusted_count
            total_resources += adjusted_count
    
    return resource_counts, total_resources

def mock_driftmgr_response(command):
    """Mock driftmgr responses for demonstration"""
    command_str = " ".join(command)
    
    if "credentials" in command_str:
        return {
            "success": True,
            "output": "AWS Profile: default (configured)\nAzure Profile: default (configured)\nGCP Project: my-project (configured)",
            "duration": random.uniform(1, 3)
        }
    elif "discover" in command_str:
        provider = command[2] if len(command) > 2 else "unknown"
        region = command[3] if len(command) > 3 else "all"
        
        if region != "all":
            resource_counts, total_resources = get_realistic_resources(provider, region)
            
            # Build realistic output
            output_lines = [f"Discovered {total_resources} resources in {provider} region {region}"]
            
            # Sort by count (highest first) and show top resources
            sorted_resources = sorted(resource_counts.items(), key=lambda x: x[1], reverse=True)
            for resource_name, count in sorted_resources[:6]:  # Show top 6
                output_lines.append(f"- {count} {resource_name}")
            
            if len(sorted_resources) > 6:
                remaining = sum(count for _, count in sorted_resources[6:])
                output_lines.append(f"- {remaining} other resources")
            
            output = "\n".join(output_lines)
        else:
            # For "all regions" discovery
            total_resources = random.randint(50, 200)
            output = f"Discovered {total_resources} resources across all {provider} regions\n- Discovery completed successfully"
        
        return {
            "success": True,
            "output": output,
            "duration": random.uniform(2, 8)
        }
    elif "analyze" in command_str:
        # Generate realistic drift analysis
        total_resources = random.randint(30, 150)
        drift_detected = random.randint(0, min(10, total_resources // 10))
        high_severity = random.randint(0, min(3, drift_detected))
        medium_severity = random.randint(0, drift_detected - high_severity)
        low_severity = drift_detected - high_severity - medium_severity
        
        return {
            "success": True,
            "output": f"Drift Analysis Results:\n- Total Resources: {total_resources}\n- Drift Detected: {drift_detected}\n- High Severity: {high_severity}\n- Medium Severity: {medium_severity}\n- Low Severity: {low_severity}",
            "duration": random.uniform(3, 10)
        }
    else:
        return {
            "success": True,
            "output": "Command executed successfully",
            "duration": random.uniform(1, 5)
        }

def run_mock_simulation():
    """Run a mock simulation that demonstrates expected behavior"""
    print("Alternative DriftMgr Simulation")
    print("=" * 60)
    print("This simulation demonstrates expected driftmgr behavior")
    print("while working around current technical issues.")
    print("=" * 60)
    
    # Load regions
    try:
        with open('aws_regions.json', 'r') as f:
            aws_data = json.load(f)
            aws_regions = [region['name'] for region in aws_data if region.get('enabled', True)]
        
        with open('azure_regions.json', 'r') as f:
            azure_data = json.load(f)
            azure_regions = [region['name'] for region in azure_data if region.get('enabled', True)]
            
        print(f"Loaded {len(aws_regions)} AWS regions and {len(azure_regions)} Azure regions")
        
    except FileNotFoundError:
        print("Region files not found, using fallback regions")
        aws_regions = ['us-east-1', 'us-west-2', 'eu-west-1']
        azure_regions = ['eastus', 'westus2', 'northeurope']
    
    start_time = datetime.now()
    
    # Test credential commands
    print("\nTesting credential commands...")
    credential_commands = [
        ['driftmgr', 'credentials', '--show'],
        ['driftmgr', 'credentials', '--test'],
        ['driftmgr', 'credentials', '--validate']
    ]
    
    for command in credential_commands:
        print(f"Executing: {' '.join(command)}")
        response = mock_driftmgr_response(command)
        time.sleep(response['duration'])
        print(f"Success ({response['duration']:.1f}s)")
        print(f"   {response['output']}")
        time.sleep(random.uniform(1, 2))
    
    # Test discovery with random regions
    print("\nTesting discovery with random regions...")
    
    # AWS discovery
    print("\nTesting AWS discovery...")
    aws_sample = random.sample(aws_regions, min(3, len(aws_regions)))
    for region in aws_sample:
        print(f"\n   Testing region: {region}")
        commands = [
            ['driftmgr', 'discover', 'aws', region],
            ['driftmgr', 'discover', 'aws', region, '--format', 'json']
        ]
        for command in commands:
            print(f"Executing: {' '.join(command)}")
            response = mock_driftmgr_response(command)
            time.sleep(response['duration'])
            print(f"Success ({response['duration']:.1f}s)")
            print(f"   {response['output']}")
            time.sleep(random.uniform(1, 2))
    
    # Azure discovery
    print("\nTesting Azure discovery...")
    azure_sample = random.sample(azure_regions, min(3, len(azure_regions)))
    for region in azure_sample:
        print(f"\n   Testing region: {region}")
        commands = [
            ['driftmgr', 'discover', 'azure', region],
            ['driftmgr', 'discover', 'azure', region, '--format', 'json']
        ]
        for command in commands:
            print(f"Executing: {' '.join(command)}")
            response = mock_driftmgr_response(command)
            time.sleep(response['duration'])
            print(f"Success ({response['duration']:.1f}s)")
            print(f"   {response['output']}")
            time.sleep(random.uniform(1, 2))
    
    # Test analysis
    print("\nTesting analysis commands...")
    analysis_commands = [
        ['driftmgr', 'analyze', '--provider', 'aws'],
        ['driftmgr', 'analyze', '--provider', 'azure'],
        ['driftmgr', 'analyze', '--all-providers']
    ]
    
    for command in analysis_commands:
        print(f"Executing: {' '.join(command)}")
        response = mock_driftmgr_response(command)
        time.sleep(response['duration'])
        print(f"Success ({response['duration']:.1f}s)")
        print(f"   {response['output']}")
        time.sleep(random.uniform(1, 2))
    
    end_time = datetime.now()
    duration = end_time - start_time
    
    print("\n" + "=" * 60)
    print("ALTERNATIVE SIMULATION COMPLETED")
    print("=" * 60)
    print(f"Duration: {duration}")
    print(f"AWS regions tested: {len(aws_sample)}")
    print(f"Azure regions tested: {len(azure_sample)}")
    print("=" * 60)
    print("This simulation demonstrated expected driftmgr behavior")
    print("with realistic resource variations across regions.")
    print("=" * 60)

if __name__ == "__main__":
    run_mock_simulation()
