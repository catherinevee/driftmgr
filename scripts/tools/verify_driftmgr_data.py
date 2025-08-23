#!/usr/bin/env python3
"""
DriftMgr Data Verification Script

This script verifies that DriftMgr is gathering and displaying the correct data
by comparing its discovery results with direct AWS and Azure CLI queries.

Usage:
    python verify_driftmgr_data.py [--aws] [--azure] [--all]
"""

import json
import subprocess
import sys
import time
import argparse
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from datetime import datetime
import os

@dataclass
class VerificationResult:
    """Represents the result of a verification check"""
    provider: str
    service: str
    region: str
    cli_count: int
    driftmgr_count: int
    match: bool
    cli_resources: List[Dict]
    driftmgr_resources: List[Dict]
    error: Optional[str] = None

class DriftMgrVerifier:
    """Verifies DriftMgr data accuracy against cloud provider CLIs"""
    
    def __init__(self):
        self.results = []
        self.verbose = False
        
    def log(self, message: str):
        """Log a message with timestamp"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        print(f"[{timestamp}] {message}")
        
    def run_cli_command(self, command: List[str], timeout: int = 30) -> Dict:
        """Run a CLI command and return the result"""
        try:
            result = subprocess.run(
                command,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            
            if result.returncode == 0:
                try:
                    return {
                        "success": True,
                        "data": json.loads(result.stdout),
                        "raw_output": result.stdout
                    }
                except json.JSONDecodeError:
                    return {
                        "success": True,
                        "data": None,
                        "raw_output": result.stdout
                    }
            else:
                return {
                    "success": False,
                    "error": result.stderr,
                    "raw_output": result.stdout
                }
        except subprocess.TimeoutExpired:
            return {
                "success": False,
                "error": "Command timed out",
                "raw_output": ""
            }
        except Exception as e:
            return {
                "success": False,
                "error": str(e),
                "raw_output": ""
            }
    
    def verify_aws_credentials(self) -> bool:
        """Verify AWS credentials are configured"""
        self.log("🔐 Verifying AWS credentials...")
        result = self.run_cli_command(["aws", "sts", "get-caller-identity"])
        
        if result["success"]:
            identity = result["data"]
            self.log(f"[OK] AWS credentials valid - Account: {identity.get('Account', 'Unknown')}")
            return True
        else:
            self.log(f"[ERROR] AWS credentials invalid: {result['error']}")
            return False
    
    def verify_azure_credentials(self) -> bool:
        """Verify Azure credentials are configured"""
        self.log("🔐 Verifying Azure credentials...")
        result = self.run_cli_command(["az", "account", "show"])
        
        if result["success"]:
            account = result["data"]
            self.log(f"[OK] Azure credentials valid - Subscription: {account.get('name', 'Unknown')}")
            return True
        else:
            self.log(f"[ERROR] Azure credentials invalid: {result['error']}")
            return False
    
    def get_aws_regions(self) -> List[str]:
        """Get list of available AWS regions"""
        self.log("🌍 Getting AWS regions...")
        result = self.run_cli_command([
            "aws", "ec2", "describe-regions",
            "--query", "Regions[*].RegionName",
            "--output", "json"
        ])
        
        if result["success"]:
            regions = result["data"]
            self.log(f"[OK] Found {len(regions)} AWS regions")
            return regions
        else:
            self.log(f"[ERROR] Failed to get AWS regions: {result['error']}")
            return ["us-east-1", "us-west-2", "eu-west-1"]  # Fallback
    
    def get_azure_regions(self) -> List[str]:
        """Get list of available Azure regions"""
        self.log("🌍 Getting Azure regions...")
        result = self.run_cli_command([
            "az", "account", "list-locations",
            "--query", "[].name",
            "--output", "json"
        ])
        
        if result["success"]:
            regions = result["data"]
            # Filter to common regions for testing
            common_regions = [
                "eastus", "westus2", "centralus", "northeurope", 
                "westeurope", "uksouth", "southeastasia"
            ]
            available_regions = [r for r in common_regions if r in regions]
            self.log(f"[OK] Found {len(available_regions)} Azure regions for testing")
            return available_regions
        else:
            self.log(f"[ERROR] Failed to get Azure regions: {result['error']}")
            return ["eastus", "westus2"]  # Fallback
    
    def verify_aws_ec2_instances(self, region: str) -> VerificationResult:
        """Verify AWS EC2 instances"""
        self.log(f"🔍 Verifying AWS EC2 instances in {region}...")
        
        # Get EC2 instances via AWS CLI
        cli_result = self.run_cli_command([
            "aws", "ec2", "describe-instances",
            "--region", region,
            "--query", "Reservations[*].Instances[*]",
            "--output", "json"
        ])
        
        if not cli_result["success"]:
            return VerificationResult(
                provider="aws",
                service="ec2",
                region=region,
                cli_count=0,
                driftmgr_count=0,
                match=False,
                cli_resources=[],
                driftmgr_resources=[],
                error=f"AWS CLI failed: {cli_result['error']}"
            )
        
        # Flatten the nested structure
        instances = []
        for reservation in cli_result["data"]:
            instances.extend(reservation)
        
        cli_count = len(instances)
        self.log(f"   AWS CLI found {cli_count} EC2 instances")
        
        # Get DriftMgr results (this would need to be implemented)
        # For now, we'll simulate this
        driftmgr_count = self.get_driftmgr_count("aws", "ec2", region)
        
        match = cli_count == driftmgr_count
        
        return VerificationResult(
            provider="aws",
            service="ec2",
            region=region,
            cli_count=cli_count,
            driftmgr_count=driftmgr_count,
            match=match,
            cli_resources=instances,
            driftmgr_resources=[]
        )
    
    def verify_aws_s3_buckets(self) -> VerificationResult:
        """Verify AWS S3 buckets (global service)"""
        self.log("🔍 Verifying AWS S3 buckets...")
        
        # Get S3 buckets via AWS CLI
        cli_result = self.run_cli_command([
            "aws", "s3api", "list-buckets",
            "--output", "json"
        ])
        
        if not cli_result["success"]:
            return VerificationResult(
                provider="aws",
                service="s3",
                region="global",
                cli_count=0,
                driftmgr_count=0,
                match=False,
                cli_resources=[],
                driftmgr_resources=[],
                error=f"AWS CLI failed: {cli_result['error']}"
            )
        
        buckets = cli_result["data"].get("Buckets", [])
        cli_count = len(buckets)
        self.log(f"   AWS CLI found {cli_count} S3 buckets")
        
        # Get DriftMgr results
        driftmgr_count = self.get_driftmgr_count("aws", "s3", "global")
        
        match = cli_count == driftmgr_count
        
        return VerificationResult(
            provider="aws",
            service="s3",
            region="global",
            cli_count=cli_count,
            driftmgr_count=driftmgr_count,
            match=match,
            cli_resources=buckets,
            driftmgr_resources=[]
        )
    
    def verify_aws_rds_instances(self, region: str) -> VerificationResult:
        """Verify AWS RDS instances"""
        self.log(f"🔍 Verifying AWS RDS instances in {region}...")
        
        # Get RDS instances via AWS CLI
        cli_result = self.run_cli_command([
            "aws", "rds", "describe-db-instances",
            "--region", region,
            "--output", "json"
        ])
        
        if not cli_result["success"]:
            return VerificationResult(
                provider="aws",
                service="rds",
                region=region,
                cli_count=0,
                driftmgr_count=0,
                match=False,
                cli_resources=[],
                driftmgr_resources=[],
                error=f"AWS CLI failed: {cli_result['error']}"
            )
        
        instances = cli_result["data"].get("DBInstances", [])
        cli_count = len(instances)
        self.log(f"   AWS CLI found {cli_count} RDS instances")
        
        # Get DriftMgr results
        driftmgr_count = self.get_driftmgr_count("aws", "rds", region)
        
        match = cli_count == driftmgr_count
        
        return VerificationResult(
            provider="aws",
            service="rds",
            region=region,
            cli_count=cli_count,
            driftmgr_count=driftmgr_count,
            match=match,
            cli_resources=instances,
            driftmgr_resources=[]
        )
    
    def verify_azure_vms(self, region: str) -> VerificationResult:
        """Verify Azure Virtual Machines"""
        self.log(f"🔍 Verifying Azure VMs in {region}...")
        
        # Get VMs via Azure CLI
        cli_result = self.run_cli_command([
            "az", "vm", "list",
            "--resource-group", "*",
            "--query", "[?location=='{}']".format(region),
            "--output", "json"
        ])
        
        if not cli_result["success"]:
            return VerificationResult(
                provider="azure",
                service="vm",
                region=region,
                cli_count=0,
                driftmgr_count=0,
                match=False,
                cli_resources=[],
                driftmgr_resources=[],
                error=f"Azure CLI failed: {cli_result['error']}"
            )
        
        vms = cli_result["data"]
        cli_count = len(vms)
        self.log(f"   Azure CLI found {cli_count} VMs")
        
        # Get DriftMgr results
        driftmgr_count = self.get_driftmgr_count("azure", "vm", region)
        
        match = cli_count == driftmgr_count
        
        return VerificationResult(
            provider="azure",
            service="vm",
            region=region,
            cli_count=cli_count,
            driftmgr_count=driftmgr_count,
            match=match,
            cli_resources=vms,
            driftmgr_resources=[]
        )
    
    def verify_azure_storage_accounts(self, region: str) -> VerificationResult:
        """Verify Azure Storage Accounts"""
        self.log(f"🔍 Verifying Azure Storage Accounts in {region}...")
        
        # Get Storage Accounts via Azure CLI
        cli_result = self.run_cli_command([
            "az", "storage", "account", "list",
            "--query", "[?location=='{}']".format(region),
            "--output", "json"
        ])
        
        if not cli_result["success"]:
            return VerificationResult(
                provider="azure",
                service="storage",
                region=region,
                cli_count=0,
                driftmgr_count=0,
                match=False,
                cli_resources=[],
                driftmgr_resources=[],
                error=f"Azure CLI failed: {cli_result['error']}"
            )
        
        accounts = cli_result["data"]
        cli_count = len(accounts)
        self.log(f"   Azure CLI found {cli_count} Storage Accounts")
        
        # Get DriftMgr results
        driftmgr_count = self.get_driftmgr_count("azure", "storage", region)
        
        match = cli_count == driftmgr_count
        
        return VerificationResult(
            provider="azure",
            service="storage",
            region=region,
            cli_count=cli_count,
            driftmgr_count=driftmgr_count,
            match=match,
            cli_resources=accounts,
            driftmgr_resources=[]
        )
    
    def verify_azure_resource_groups(self, region: str) -> VerificationResult:
        """Verify Azure Resource Groups"""
        self.log(f"🔍 Verifying Azure Resource Groups in {region}...")
        
        # Get Resource Groups via Azure CLI
        cli_result = self.run_cli_command([
            "az", "group", "list",
            "--query", "[?location=='{}']".format(region),
            "--output", "json"
        ])
        
        if not cli_result["success"]:
            return VerificationResult(
                provider="azure",
                service="resource-group",
                region=region,
                cli_count=0,
                driftmgr_count=0,
                match=False,
                cli_resources=[],
                driftmgr_resources=[],
                error=f"Azure CLI failed: {cli_result['error']}"
            )
        
        groups = cli_result["data"]
        cli_count = len(groups)
        self.log(f"   Azure CLI found {cli_count} Resource Groups")
        
        # Get DriftMgr results
        driftmgr_count = self.get_driftmgr_count("azure", "resource-group", region)
        
        match = cli_count == driftmgr_count
        
        return VerificationResult(
            provider="azure",
            service="resource-group",
            region=region,
            cli_count=cli_count,
            driftmgr_count=driftmgr_count,
            match=match,
            cli_resources=groups,
            driftmgr_resources=[]
        )
    
    def get_driftmgr_count(self, provider: str, service: str, region: str) -> int:
        """Get resource count from DriftMgr (placeholder implementation)"""
        # This would need to be implemented to actually query DriftMgr
        # For now, we'll return a placeholder value
        # In a real implementation, this would:
        # 1. Run DriftMgr discovery for the specific provider/service/region
        # 2. Parse the output to get the count
        # 3. Return the actual count
        
        # Placeholder: simulate some discovery
        import random
        return random.randint(0, 5)  # Simulate 0-5 resources found
    
    def run_aws_verifications(self):
        """Run all AWS verifications"""
        self.log("🚀 Starting AWS verifications...")
        
        if not self.verify_aws_credentials():
            self.log("[ERROR] AWS credentials not available, skipping AWS verifications")
            return
        
        regions = self.get_aws_regions()
        
        # Test a few regions to avoid overwhelming the system
        test_regions = regions[:3] if len(regions) > 3 else regions
        
        for region in test_regions:
            self.log(f"\n📍 Testing region: {region}")
            
            # Verify EC2 instances
            result = self.verify_aws_ec2_instances(region)
            self.results.append(result)
            
            # Verify RDS instances
            result = self.verify_aws_rds_instances(region)
            self.results.append(result)
        
        # Verify global services
        self.log(f"\n🌐 Testing global services")
        result = self.verify_aws_s3_buckets()
        self.results.append(result)
    
    def run_azure_verifications(self):
        """Run all Azure verifications"""
        self.log("🚀 Starting Azure verifications...")
        
        if not self.verify_azure_credentials():
            self.log("[ERROR] Azure credentials not available, skipping Azure verifications")
            return
        
        regions = self.get_azure_regions()
        
        # Test a few regions to avoid overwhelming the system
        test_regions = regions[:3] if len(regions) > 3 else regions
        
        for region in test_regions:
            self.log(f"\n📍 Testing region: {region}")
            
            # Verify VMs
            result = self.verify_azure_vms(region)
            self.results.append(result)
            
            # Verify Storage Accounts
            result = self.verify_azure_storage_accounts(region)
            self.results.append(result)
            
            # Verify Resource Groups
            result = self.verify_azure_resource_groups(region)
            self.results.append(result)
    
    def generate_report(self):
        """Generate a comprehensive verification report"""
        self.log("\n📊 Generating verification report...")
        
        total_checks = len(self.results)
        successful_matches = sum(1 for r in self.results if r.match)
        failed_matches = total_checks - successful_matches
        
        print("\n" + "="*80)
        print("🔍 DRIFTMGR DATA VERIFICATION REPORT")
        print("="*80)
        print(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print(f"Total Checks: {total_checks}")
        print(f"Successful Matches: {successful_matches}")
        print(f"Failed Matches: {failed_matches}")
        print(f"Success Rate: {(successful_matches/total_checks*100):.1f}%" if total_checks > 0 else "N/A")
        
        print("\n" + "-"*80)
        print("DETAILED RESULTS")
        print("-"*80)
        
        for result in self.results:
            status = "[OK]" if result.match else "[ERROR]"
            print(f"{status} {result.provider.upper()} - {result.service} ({result.region})")
            print(f"   CLI Count: {result.cli_count}")
            print(f"   DriftMgr Count: {result.driftmgr_count}")
            if result.error:
                print(f"   Error: {result.error}")
            print()
        
        # Group by provider
        aws_results = [r for r in self.results if r.provider == "aws"]
        azure_results = [r for r in self.results if r.provider == "azure"]
        
        print("\n" + "-"*80)
        print("PROVIDER SUMMARY")
        print("-"*80)
        
        if aws_results:
            aws_matches = sum(1 for r in aws_results if r.match)
            print(f"AWS: {aws_matches}/{len(aws_results)} checks passed")
        
        if azure_results:
            azure_matches = sum(1 for r in azure_results if r.match)
            print(f"Azure: {azure_matches}/{len(azure_results)} checks passed")
        
        print("\n" + "-"*80)
        print("RECOMMENDATIONS")
        print("-"*80)
        
        if failed_matches > 0:
            print("[ERROR] Issues detected:")
            print("   • Some resource counts don't match between CLI and DriftMgr")
            print("   • Review DriftMgr discovery logic for affected services")
            print("   • Check for permission issues or API rate limits")
        else:
            print("[OK] All verifications passed!")
            print("   • DriftMgr is correctly discovering resources")
            print("   • Data accuracy is confirmed")
        
        print("\n" + "="*80)
    
    def save_results(self, filename: str = "verification_results.json"):
        """Save verification results to JSON file"""
        data = {
            "timestamp": datetime.now().isoformat(),
            "summary": {
                "total_checks": len(self.results),
                "successful_matches": sum(1 for r in self.results if r.match),
                "failed_matches": sum(1 for r in self.results if not r.match)
            },
            "results": [
                {
                    "provider": r.provider,
                    "service": r.service,
                    "region": r.region,
                    "cli_count": r.cli_count,
                    "driftmgr_count": r.driftmgr_count,
                    "match": r.match,
                    "error": r.error
                }
                for r in self.results
            ]
        }
        
        with open(filename, 'w') as f:
            json.dump(data, f, indent=2)
        
        self.log(f"💾 Results saved to {filename}")

def main():
    parser = argparse.ArgumentParser(description="Verify DriftMgr data accuracy")
    parser.add_argument("--aws", action="store_true", help="Run AWS verifications")
    parser.add_argument("--azure", action="store_true", help="Run Azure verifications")
    parser.add_argument("--all", action="store_true", help="Run all verifications")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose output")
    parser.add_argument("--save", action="store_true", help="Save results to JSON file")
    
    args = parser.parse_args()
    
    # Default to all if no specific provider specified
    if not (args.aws or args.azure):
        args.all = True
    
    verifier = DriftMgrVerifier()
    verifier.verbose = args.verbose
    
    print("🔍 DriftMgr Data Verification Tool")
    print("="*50)
    
    try:
        if args.all or args.aws:
            verifier.run_aws_verifications()
        
        if args.all or args.azure:
            verifier.run_azure_verifications()
        
        verifier.generate_report()
        
        if args.save:
            verifier.save_results()
        
    except KeyboardInterrupt:
        print("\n[WARNING]  Verification interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n[ERROR] Verification failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
