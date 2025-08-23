#!/usr/bin/env python3
"""
DriftMgr CLI Verification Script

This script runs DriftMgr discovery and compares the results with direct
AWS and Azure CLI queries to validate data accuracy.

Usage:
    python verify_driftmgr_cli.py [--aws] [--azure] [--all]
"""

import json
import subprocess
import sys
import time
import argparse
import re
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

class DriftMgrCLIVerifier:
    """Verifies DriftMgr CLI output against cloud provider CLIs"""
    
    def __init__(self):
        self.results = []
        self.verbose = False
        self.driftmgr_path = "./driftmgr.exe"  # Adjust path as needed
        
    def log(self, message: str):
        """Log a message with timestamp"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        print(f"[{timestamp}] {message}")
        
    def run_command(self, command: List[str], timeout: int = 60) -> Dict:
        """Run a command and return the result"""
        try:
            result = subprocess.run(
                command,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            
            return {
                "success": result.returncode == 0,
                "stdout": result.stdout,
                "stderr": result.stderr,
                "returncode": result.returncode
            }
        except subprocess.TimeoutExpired:
            return {
                "success": False,
                "stdout": "",
                "stderr": "Command timed out",
                "returncode": -1
            }
        except Exception as e:
            return {
                "success": False,
                "stdout": "",
                "stderr": str(e),
                "returncode": -1
            }
    
    def verify_aws_credentials(self) -> bool:
        """Verify AWS credentials are configured"""
        self.log("🔐 Verifying AWS credentials...")
        result = self.run_command(["aws", "sts", "get-caller-identity"])
        
        if result["success"]:
            try:
                identity = json.loads(result["stdout"])
                self.log(f"[OK] AWS credentials valid - Account: {identity.get('Account', 'Unknown')}")
                return True
            except json.JSONDecodeError:
                self.log("[OK] AWS credentials valid (non-JSON output)")
                return True
        else:
            self.log(f"[ERROR] AWS credentials invalid: {result['stderr']}")
            return False
    
    def verify_azure_credentials(self) -> bool:
        """Verify Azure credentials are configured"""
        self.log("🔐 Verifying Azure credentials...")
        result = self.run_command(["az", "account", "show"])
        
        if result["success"]:
            try:
                account = json.loads(result["stdout"])
                self.log(f"[OK] Azure credentials valid - Subscription: {account.get('name', 'Unknown')}")
                return True
            except json.JSONDecodeError:
                self.log("[OK] Azure credentials valid (non-JSON output)")
                return True
        else:
            self.log(f"[ERROR] Azure credentials invalid: {result['stderr']}")
            return False
    
    def run_driftmgr_discovery(self, provider: str, regions: List[str] = None) -> Dict:
        """Run DriftMgr discovery for a specific provider"""
        self.log(f"🚀 Running DriftMgr discovery for {provider}...")
        
        # Build the command
        cmd = [self.driftmgr_path, "discover", "--provider", provider]
        
        if regions:
            cmd.extend(["--regions"] + regions)
        
        if self.verbose:
            cmd.append("--verbose")
        
        self.log(f"   Command: {' '.join(cmd)}")
        
        result = self.run_command(cmd, timeout=300)  # 5 minutes timeout
        
        if result["success"]:
            self.log(f"[OK] DriftMgr discovery completed for {provider}")
            return result
        else:
            self.log(f"[ERROR] DriftMgr discovery failed for {provider}: {result['stderr']}")
            return result
    
    def parse_driftmgr_output(self, output: str) -> Dict[str, int]:
        """Parse DriftMgr output to extract resource counts"""
        resource_counts = {}
        
        # Look for patterns like "Found X resources" or "Discovered X Y"
        patterns = [
            r"Found (\d+) (\w+)",
            r"Discovered (\d+) (\w+)",
            r"(\d+) (\w+) found",
            r"(\d+) (\w+) discovered"
        ]
        
        lines = output.split('\n')
        for line in lines:
            for pattern in patterns:
                match = re.search(pattern, line, re.IGNORECASE)
                if match:
                    count = int(match.group(1))
                    resource_type = match.group(2).lower()
                    resource_counts[resource_type] = count
        
        return resource_counts
    
    def get_aws_cli_counts(self, regions: List[str]) -> Dict[str, int]:
        """Get resource counts from AWS CLI"""
        counts = {}
        
        # Get S3 buckets (global)
        self.log("   Getting S3 bucket count...")
        result = self.run_command(["aws", "s3api", "list-buckets"])
        if result["success"]:
            try:
                data = json.loads(result["stdout"])
                counts["s3"] = len(data.get("Buckets", []))
            except json.JSONDecodeError:
                counts["s3"] = 0
        
        # Get EC2 instances per region
        for region in regions[:3]:  # Limit to first 3 regions
            self.log(f"   Getting EC2 instances in {region}...")
            result = self.run_command([
                "aws", "ec2", "describe-instances",
                "--region", region,
                "--query", "Reservations[*].Instances[*]"
            ])
            if result["success"]:
                try:
                    data = json.loads(result["stdout"])
                    # Flatten nested structure
                    instances = []
                    for reservation in data:
                        instances.extend(reservation)
                    counts[f"ec2_{region}"] = len(instances)
                except json.JSONDecodeError:
                    counts[f"ec2_{region}"] = 0
        
        return counts
    
    def get_azure_cli_counts(self, regions: List[str]) -> Dict[str, int]:
        """Get resource counts from Azure CLI"""
        counts = {}
        
        for region in regions[:3]:  # Limit to first 3 regions
            # Get VMs
            self.log(f"   Getting VMs in {region}...")
            result = self.run_command([
                "az", "vm", "list",
                "--resource-group", "*",
                "--query", f"[?location=='{region}']"
            ])
            if result["success"]:
                try:
                    data = json.loads(result["stdout"])
                    counts[f"vm_{region}"] = len(data)
                except json.JSONDecodeError:
                    counts[f"vm_{region}"] = 0
            
            # Get Storage Accounts
            self.log(f"   Getting Storage Accounts in {region}...")
            result = self.run_command([
                "az", "storage", "account", "list",
                "--query", f"[?location=='{region}']"
            ])
            if result["success"]:
                try:
                    data = json.loads(result["stdout"])
                    counts[f"storage_{region}"] = len(data)
                except json.JSONDecodeError:
                    counts[f"storage_{region}"] = 0
        
        return counts
    
    def verify_aws_discovery(self):
        """Verify AWS discovery results"""
        self.log("🔍 Starting AWS discovery verification...")
        
        if not self.verify_aws_credentials():
            self.log("[ERROR] AWS credentials not available, skipping AWS verification")
            return
        
        # Get AWS regions
        self.log("🌍 Getting AWS regions...")
        result = self.run_command([
            "aws", "ec2", "describe-regions",
            "--query", "Regions[*].RegionName",
            "--output", "json"
        ])
        
        if result["success"]:
            try:
                regions = json.loads(result["stdout"])
                test_regions = regions[:3]  # Test first 3 regions
                self.log(f"[OK] Testing {len(test_regions)} AWS regions: {', '.join(test_regions)}")
            except json.JSONDecodeError:
                test_regions = ["us-east-1", "us-west-2"]
                self.log(f"[WARNING]  Using fallback regions: {', '.join(test_regions)}")
        else:
            test_regions = ["us-east-1", "us-west-2"]
            self.log(f"[WARNING]  Using fallback regions: {', '.join(test_regions)}")
        
        # Run DriftMgr discovery
        driftmgr_result = self.run_driftmgr_discovery("aws", test_regions)
        
        if not driftmgr_result["success"]:
            self.log("[ERROR] DriftMgr AWS discovery failed")
            return
        
        # Parse DriftMgr output
        driftmgr_counts = self.parse_driftmgr_output(driftmgr_result["stdout"])
        self.log(f"   DriftMgr found: {driftmgr_counts}")
        
        # Get CLI counts
        cli_counts = self.get_aws_cli_counts(test_regions)
        self.log(f"   AWS CLI found: {cli_counts}")
        
        # Compare results
        self.compare_counts("aws", driftmgr_counts, cli_counts)
    
    def verify_azure_discovery(self):
        """Verify Azure discovery results"""
        self.log("🔍 Starting Azure discovery verification...")
        
        if not self.verify_azure_credentials():
            self.log("[ERROR] Azure credentials not available, skipping Azure verification")
            return
        
        # Use common Azure regions for testing
        test_regions = ["eastus", "westus2", "centralus"]
        self.log(f"[OK] Testing {len(test_regions)} Azure regions: {', '.join(test_regions)}")
        
        # Run DriftMgr discovery
        driftmgr_result = self.run_driftmgr_discovery("azure", test_regions)
        
        if not driftmgr_result["success"]:
            self.log("[ERROR] DriftMgr Azure discovery failed")
            return
        
        # Parse DriftMgr output
        driftmgr_counts = self.parse_driftmgr_output(driftmgr_result["stdout"])
        self.log(f"   DriftMgr found: {driftmgr_counts}")
        
        # Get CLI counts
        cli_counts = self.get_azure_cli_counts(test_regions)
        self.log(f"   Azure CLI found: {cli_counts}")
        
        # Compare results
        self.compare_counts("azure", driftmgr_counts, cli_counts)
    
    def compare_counts(self, provider: str, driftmgr_counts: Dict[str, int], cli_counts: Dict[str, int]):
        """Compare DriftMgr counts with CLI counts"""
        self.log(f"\n📊 Comparing {provider.upper()} resource counts...")
        
        all_keys = set(driftmgr_counts.keys()) | set(cli_counts.keys())
        
        for key in all_keys:
            driftmgr_count = driftmgr_counts.get(key, 0)
            cli_count = cli_counts.get(key, 0)
            match = driftmgr_count == cli_count
            
            status = "[OK]" if match else "[ERROR]"
            print(f"{status} {key}: DriftMgr={driftmgr_count}, CLI={cli_count}")
            
            # Store result
            result = VerificationResult(
                provider=provider,
                service=key,
                region="multiple",
                cli_count=cli_count,
                driftmgr_count=driftmgr_count,
                match=match,
                cli_resources=[],
                driftmgr_resources=[]
            )
            self.results.append(result)
    
    def run_quick_test(self):
        """Run a quick test to verify DriftMgr is working"""
        self.log("[LIGHTNING] Running quick DriftMgr test...")
        
        # Test if DriftMgr executable exists
        if not os.path.exists(self.driftmgr_path):
            self.log(f"[ERROR] DriftMgr executable not found at {self.driftmgr_path}")
            self.log("   Please ensure DriftMgr is built and the path is correct")
            return False
        
        # Test DriftMgr help
        result = self.run_command([self.driftmgr_path, "--help"])
        if result["success"]:
            self.log("[OK] DriftMgr executable is working")
            return True
        else:
            self.log(f"[ERROR] DriftMgr executable failed: {result['stderr']}")
            return False
    
    def generate_report(self):
        """Generate a verification report"""
        self.log("\n📊 Generating verification report...")
        
        total_checks = len(self.results)
        successful_matches = sum(1 for r in self.results if r.match)
        failed_matches = total_checks - successful_matches
        
        print("\n" + "="*80)
        print("🔍 DRIFTMGR CLI VERIFICATION REPORT")
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
            print(f"{status} {result.provider.upper()} - {result.service}")
            print(f"   DriftMgr Count: {result.driftmgr_count}")
            print(f"   CLI Count: {result.cli_count}")
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
            print("   • Some resource counts don't match between DriftMgr and CLI")
            print("   • Review DriftMgr discovery logic for affected services")
            print("   • Check for permission issues or API rate limits")
            print("   • Verify DriftMgr configuration and credentials")
        else:
            print("[OK] All verifications passed!")
            print("   • DriftMgr is correctly discovering resources")
            print("   • Data accuracy is confirmed")
        
        print("\n" + "="*80)
    
    def save_results(self, filename: str = "driftmgr_verification_results.json"):
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
    parser = argparse.ArgumentParser(description="Verify DriftMgr CLI data accuracy")
    parser.add_argument("--aws", action="store_true", help="Run AWS verifications")
    parser.add_argument("--azure", action="store_true", help="Run Azure verifications")
    parser.add_argument("--all", action="store_true", help="Run all verifications")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose output")
    parser.add_argument("--save", action="store_true", help="Save results to JSON file")
    parser.add_argument("--driftmgr-path", default="./driftmgr.exe", help="Path to DriftMgr executable")
    
    args = parser.parse_args()
    
    # Default to all if no specific provider specified
    if not (args.aws or args.azure):
        args.all = True
    
    verifier = DriftMgrCLIVerifier()
    verifier.verbose = args.verbose
    verifier.driftmgr_path = args.driftmgr_path
    
    print("🔍 DriftMgr CLI Verification Tool")
    print("="*50)
    
    try:
        # Run quick test first
        if not verifier.run_quick_test():
            print("[ERROR] DriftMgr is not working properly. Please check the installation.")
            sys.exit(1)
        
        if args.all or args.aws:
            verifier.verify_aws_discovery()
        
        if args.all or args.azure:
            verifier.verify_azure_discovery()
        
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
