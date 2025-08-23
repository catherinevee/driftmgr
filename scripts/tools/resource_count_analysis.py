#!/usr/bin/env python3
"""
DriftMgr Resource Count Analysis

This script analyzes and counts all resources detected by driftmgr across all providers,
accounts, and regions to provide a comprehensive resource inventory.
"""

import json
import subprocess
import time
import sys
from datetime import datetime

def safe_emoji(emoji_code):
    """Safely return emoji or fallback text"""
    try:
        return emoji_code
    except UnicodeEncodeError:
        emoji_map = {
            "[OK]": "[PASS]",
            "[ERROR]": "[FAIL]", 
            "💥": "[ERROR]",
            "⏰": "[TIMEOUT]",
            "🔍": "[SEARCH]",
            "🌍": "[GLOBAL]",
            "📊": "[REPORT]",
            "📁": "[FOLDER]"
        }
        return emoji_map.get(emoji_code, "[INFO]")

class ResourceAnalyzer:
    def __init__(self):
        self.results = {
            'aws': {'accounts': [], 'total_resources': 0, 'regions_tested': []},
            'azure': {'accounts': [], 'total_resources': 0, 'regions_tested': []},
            'gcp': {'accounts': [], 'total_resources': 0, 'regions_tested': []},
            'digitalocean': {'accounts': [], 'total_resources': 0, 'regions_tested': []}
        }
        self.total_resources = 0
        self.total_accounts = 0
        
    def run_command(self, command, timeout=60):
        """Run a driftmgr command and return the result"""
        try:
            print(f"\n{safe_emoji('🔍')} Executing: {' '.join(command)}")
            start_time = time.time()
            
            result = subprocess.run(
                command,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            
            duration = time.time() - start_time
            
            if result.returncode == 0:
                print(f"{safe_emoji('[OK]')} Command completed successfully ({duration:.2f}s)")
                return {
                    'command': ' '.join(command),
                    'return_code': result.returncode,
                    'stdout': result.stdout,
                    'stderr': result.stderr,
                    'duration': duration,
                    'success': True
                }
            else:
                print(f"{safe_emoji('[ERROR]')} Command failed ({duration:.2f}s)")
                if result.stderr.strip():
                    print(f"   Error: {result.stderr.strip()[:100]}...")
                return {
                    'command': ' '.join(command),
                    'return_code': result.returncode,
                    'stdout': result.stdout,
                    'stderr': result.stderr,
                    'duration': duration,
                    'success': False
                }
                
        except subprocess.TimeoutExpired:
            print(f"{safe_emoji('⏰')} Command timed out after {timeout}s")
            return {
                'command': ' '.join(command),
                'return_code': -1,
                'stdout': '',
                'stderr': 'Command timed out',
                'duration': timeout,
                'success': False
            }
        except Exception as e:
            print(f"{safe_emoji('💥')} Error: {e}")
            return {
                'command': ' '.join(command),
                'return_code': -1,
                'stdout': '',
                'stderr': str(e),
                'duration': 0,
                'success': False
            }
    
    def get_available_accounts(self):
        """Get all available accounts across providers"""
        print(f"\n{'='*60}")
        print(f"{safe_emoji('📊')} Getting Available Accounts")
        print(f"{'='*60}")
        
        result = self.run_command(['driftmgr', 'credentials', 'accounts'])
        
        if result['success']:
            # Parse the output to extract account information
            lines = result['stdout'].split('\n')
            current_provider = None
            
            for line in lines:
                line = line.strip()
                if line.startswith('AWS:'):
                    current_provider = 'aws'
                elif line.startswith('Azure:'):
                    current_provider = 'azure'
                elif line.startswith('GCP:'):
                    current_provider = 'gcp'
                elif line.startswith('DigitalOcean:'):
                    current_provider = 'digitalocean'
                elif line.startswith('✓') and current_provider:
                    # Parse account line like: "✓ Account 025066254478 (ACTIVE)"
                    parts = line.split()
                    if len(parts) >= 3:
                        account_name = parts[1]
                        account_id = parts[2].strip('()')
                        self.results[current_provider]['accounts'].append({
                            'name': account_name,
                            'id': account_id,
                            'status': 'ACTIVE' if '(ACTIVE)' in line else 'INACTIVE'
                        })
        
        # Print summary
        for provider, data in self.results.items():
            if data['accounts']:
                print(f"\n{provider.upper()}: {len(data['accounts'])} accounts")
                for account in data['accounts']:
                    print(f"  - {account['name']} ({account['id']}) - {account['status']}")
    
    def discover_resources_in_region(self, provider, region):
        """Discover resources in a specific region for a provider"""
        print(f"\n{safe_emoji('🔍')} Discovering {provider.upper()} resources in {region}...")
        
        result = self.run_command(['driftmgr', 'discover', provider, region])
        
        if result['success']:
            # Parse the output to extract resource count
            output = result['stdout']
            
            # Look for resource count in the output
            if 'Found 0 resources' in output:
                resource_count = 0
            elif 'Found' in output and 'resources' in output:
                # Try to extract number from "Found X resources"
                import re
                match = re.search(r'Found (\d+) resources', output)
                if match:
                    resource_count = int(match.group(1))
                else:
                    resource_count = 0
            else:
                resource_count = 0
            
            self.results[provider]['total_resources'] += resource_count
            self.results[provider]['regions_tested'].append(region)
            
            print(f"   {safe_emoji('📊')} Found {resource_count} resources in {region}")
            return resource_count
        else:
            print(f"   {safe_emoji('[ERROR]')} Failed to discover resources in {region}")
            return 0
    
    def analyze_aws_resources(self):
        """Analyze AWS resources across multiple regions"""
        print(f"\n{'='*60}")
        print(f"{safe_emoji('🌍')} Analyzing AWS Resources")
        print(f"{'='*60}")
        
        # Test multiple AWS regions
        aws_regions = [
            'us-east-1', 'us-west-2', 'us-west-1', 'eu-west-1', 'eu-central-1',
            'ap-southeast-1', 'ap-southeast-2', 'ap-northeast-1', 'ca-central-1'
        ]
        
        for region in aws_regions:
            self.discover_resources_in_region('aws', region)
            time.sleep(1)  # Brief pause between requests
    
    def analyze_azure_resources(self):
        """Analyze Azure resources across multiple regions"""
        print(f"\n{'='*60}")
        print(f"{safe_emoji('🌍')} Analyzing Azure Resources")
        print(f"{'='*60}")
        
        # Test multiple Azure regions
        azure_regions = [
            'eastus', 'westus2', 'northeurope', 'westeurope', 'southeastasia',
            'australiaeast', 'centralus', 'southcentralus'
        ]
        
        for region in azure_regions:
            self.discover_resources_in_region('azure', region)
            time.sleep(1)  # Brief pause between requests
    
    def analyze_gcp_resources(self):
        """Analyze GCP resources across multiple regions"""
        print(f"\n{'='*60}")
        print(f"{safe_emoji('🌍')} Analyzing GCP Resources")
        print(f"{'='*60}")
        
        # Test multiple GCP regions
        gcp_regions = [
            'us-central1', 'us-east1', 'us-west1', 'europe-west1', 'asia-southeast1',
            'australia-southeast1', 'northamerica-northeast1'
        ]
        
        for region in gcp_regions:
            self.discover_resources_in_region('gcp', region)
            time.sleep(1)  # Brief pause between requests
    
    def analyze_digitalocean_resources(self):
        """Analyze DigitalOcean resources"""
        print(f"\n{'='*60}")
        print(f"{safe_emoji('🌍')} Analyzing DigitalOcean Resources")
        print(f"{'='*60}")
        
        # DigitalOcean doesn't have traditional regions like others
        result = self.run_command(['driftmgr', 'discover', 'digitalocean'])
        
        if result['success']:
            output = result['stdout']
            if 'Found 0 resources' in output:
                resource_count = 0
            elif 'Found' in output and 'resources' in output:
                import re
                match = re.search(r'Found (\d+) resources', output)
                if match:
                    resource_count = int(match.group(1))
                else:
                    resource_count = 0
            else:
                resource_count = 0
            
            self.results['digitalocean']['total_resources'] = resource_count
            print(f"   {safe_emoji('📊')} Found {resource_count} DigitalOcean resources")
        else:
            print(f"   {safe_emoji('[ERROR]')} Failed to discover DigitalOcean resources")
    
    def generate_summary_report(self):
        """Generate a comprehensive summary report"""
        print(f"\n{'='*60}")
        print(f"{safe_emoji('📊')} RESOURCE COUNT SUMMARY")
        print(f"{'='*60}")
        
        # Calculate totals
        for provider, data in self.results.items():
            self.total_resources += data['total_resources']
            self.total_accounts += len(data['accounts'])
        
        print(f"\n{safe_emoji('📊')} OVERALL SUMMARY:")
        print(f"   Total Resources: {self.total_resources}")
        print(f"   Total Accounts: {self.total_accounts}")
        print(f"   Providers Analyzed: {len([p for p, d in self.results.items() if d['accounts']])}")
        
        print(f"\n{safe_emoji('🌍')} BY PROVIDER:")
        for provider, data in self.results.items():
            if data['accounts']:
                print(f"   {provider.upper()}:")
                print(f"     Accounts: {len(data['accounts'])}")
                print(f"     Resources: {data['total_resources']}")
                print(f"     Regions Tested: {len(data['regions_tested'])}")
                if data['regions_tested']:
                    print(f"     Regions: {', '.join(data['regions_tested'])}")
        
        print(f"\n{safe_emoji('📁')} DETAILED BREAKDOWN:")
        for provider, data in self.results.items():
            if data['accounts']:
                print(f"\n   {provider.upper()}:")
                for account in data['accounts']:
                    print(f"     - {account['name']} ({account['id']}) - {account['status']}")
        
        # Save detailed report to file
        report_data = {
            'timestamp': datetime.now().isoformat(),
            'summary': {
                'total_resources': self.total_resources,
                'total_accounts': self.total_accounts,
                'providers_analyzed': len([p for p, d in self.results.items() if d['accounts']])
            },
            'detailed_results': self.results
        }
        
        with open('resource_count_report.json', 'w') as f:
            json.dump(report_data, f, indent=2)
        
        print(f"\n{safe_emoji('📁')} Detailed report saved to: resource_count_report.json")
        
        return report_data
    
    def run_analysis(self):
        """Run the complete resource analysis"""
        print("DriftMgr Resource Count Analysis")
        print("================================")
        print("This script will analyze and count all resources detected by driftmgr")
        print("across all providers, accounts, and regions.")
        print()
        
        start_time = time.time()
        
        # Get available accounts
        self.get_available_accounts()
        
        # Analyze resources for each provider
        if self.results['aws']['accounts']:
            self.analyze_aws_resources()
        
        if self.results['azure']['accounts']:
            self.analyze_azure_resources()
        
        if self.results['gcp']['accounts']:
            self.analyze_gcp_resources()
        
        if self.results['digitalocean']['accounts']:
            self.analyze_digitalocean_resources()
        
        # Generate summary report
        report = self.generate_summary_report()
        
        total_time = time.time() - start_time
        
        print(f"\n{'='*60}")
        print(f"{safe_emoji('🎉')} Analysis completed in {total_time:.2f} seconds")
        print(f"{'='*60}")
        
        return report

def main():
    """Main entry point"""
    analyzer = ResourceAnalyzer()
    report = analyzer.run_analysis()
    
    print(f"\n{safe_emoji('📊')} FINAL RESULT:")
    print(f"   DriftMgr detected {report['summary']['total_resources']} resources")
    print(f"   across {report['summary']['total_accounts']} accounts")
    print(f"   in {report['summary']['providers_analyzed']} cloud providers")

if __name__ == "__main__":
    main()
