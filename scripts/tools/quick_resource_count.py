#!/usr/bin/env python3
"""
Quick Resource Count for DriftMgr

This script quickly counts resources across multiple regions and providers.
"""

import subprocess
import re
import time

def run_discovery(provider, region):
    """Run discovery for a specific provider and region"""
    try:
        cmd = ['driftmgr', 'discover', provider, region]
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        
        if result.returncode == 0:
            # Look for resource count in output
            output = result.stdout + result.stderr
            
            # Find "Found X resources" pattern
            match = re.search(r'Found (\d+) resources', output)
            if match:
                return int(match.group(1))
            else:
                return 0
        else:
            return 0
    except:
        return 0

def main():
    print("DriftMgr Quick Resource Count")
    print("============================")
    
    # Define regions to test
    aws_regions = ['us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1']
    azure_regions = ['eastus', 'westus2', 'northeurope', 'southeastasia']
    gcp_regions = ['us-central1', 'us-east1', 'europe-west1', 'asia-southeast1']
    
    total_resources = 0
    total_regions_tested = 0
    
    print("\nTesting AWS regions...")
    for region in aws_regions:
        count = run_discovery('aws', region)
        print(f"  {region}: {count} resources")
        total_resources += count
        total_regions_tested += 1
        time.sleep(1)
    
    print("\nTesting Azure regions...")
    for region in azure_regions:
        count = run_discovery('azure', region)
        print(f"  {region}: {count} resources")
        total_resources += count
        total_regions_tested += 1
        time.sleep(1)
    
    print("\nTesting GCP regions...")
    for region in gcp_regions:
        count = run_discovery('gcp', region)
        print(f"  {region}: {count} resources")
        total_resources += count
        total_regions_tested += 1
        time.sleep(1)
    
    print(f"\n{'='*50}")
    print(f"SUMMARY:")
    print(f"  Total Resources Found: {total_resources}")
    print(f"  Regions Tested: {total_regions_tested}")
    print(f"  Average per Region: {total_resources/total_regions_tested:.1f}")
    print(f"{'='*50}")

if __name__ == "__main__":
    main()
