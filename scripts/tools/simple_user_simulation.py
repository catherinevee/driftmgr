#!/usr/bin/env python3
"""
Simple DriftMgr User Simulation

This script simulates a user using driftmgr with auto-detected credentials
and tests core features across random AWS and Azure regions.
"""

import json
import random
import subprocess
import time
import sys
from datetime import datetime

def load_regions():
    """Load region data from JSON files"""
    aws_regions = []
    azure_regions = []
    
    try:
        # Load AWS regions
        with open('aws_regions.json', 'r') as f:
            aws_data = json.load(f)
            aws_regions = [region['name'] for region in aws_data if region.get('enabled', True)]
        
        # Load Azure regions
        with open('azure_regions.json', 'r') as f:
            azure_data = json.load(f)
            azure_regions = [region['name'] for region in azure_data if region.get('enabled', True)]
            
        print(f"[OK] Loaded {len(aws_regions)} AWS regions and {len(azure_regions)} Azure regions")
        
    except FileNotFoundError as e:
        print(f"[WARNING] Region file not found: {e}")
        # Fallback to common regions
        aws_regions = ['us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1', 'ca-central-1']
        azure_regions = ['eastus', 'westus2', 'northeurope', 'southeastasia', 'uksouth']
    
    return aws_regions, azure_regions

def run_command(command, timeout=30):
    """Run a driftmgr command and return the result"""
    try:
        print(f"🔍 Executing: {' '.join(command)}")
        start_time = time.time()
        
        result = subprocess.run(
            command,
            capture_output=True,
            text=True,
            timeout=timeout
        )
        
        duration = time.time() - start_time
        
        if result.returncode == 0:
            print(f"[OK] Success ({duration:.2f}s)")
            return True
        else:
            print(f"[ERROR] Failed ({duration:.2f}s)")
            if result.stderr.strip():
                print(f"   Error: {result.stderr.strip()[:100]}...")
            return False
            
    except subprocess.TimeoutExpired:
        print(f"⏰ Timeout after {timeout}s")
        return False
    except Exception as e:
        print(f"💥 Exception: {e}")
        return False

def simulate_credential_check():
    """Simulate user checking auto-detected credentials"""
    print("\n" + "="*60)
    print("🔐 CREDENTIAL AUTO-DETECTION TEST")
    print("="*60)
    
    commands = [
        ['driftmgr', 'credentials', '--show'],
        ['driftmgr', 'credentials', '--test'],
        ['driftmgr', 'credentials', '--validate']
    ]
    
    for command in commands:
        run_command(command, timeout=15)
        time.sleep(random.uniform(1, 2))

def simulate_discovery_with_random_regions(aws_regions, azure_regions):
    """Simulate discovery using random regions"""
    print("\n" + "="*60)
    print("🌍 DISCOVERY WITH RANDOM REGIONS")
    print("="*60)
    
    # Test AWS discovery with random regions
    print("\n🔍 Testing AWS discovery...")
    aws_sample = random.sample(aws_regions, min(3, len(aws_regions)))
    for region in aws_sample:
        print(f"\n   Testing region: {region}")
        commands = [
            ['driftmgr', 'discover', 'aws', region],
            ['driftmgr', 'discover', 'aws', region, '--format', 'json']
        ]
        for command in commands:
            run_command(command, timeout=60)
            time.sleep(random.uniform(1, 3))
    
    # Test Azure discovery with random regions
    print("\n🔍 Testing Azure discovery...")
    azure_sample = random.sample(azure_regions, min(3, len(azure_regions)))
    for region in azure_sample:
        print(f"\n   Testing region: {region}")
        commands = [
            ['driftmgr', 'discover', 'azure', region],
            ['driftmgr', 'discover', 'azure', region, '--format', 'json']
        ]
        for command in commands:
            run_command(command, timeout=60)
            time.sleep(random.uniform(1, 3))

def simulate_analysis_features():
    """Simulate analysis features"""
    print("\n" + "="*60)
    print("📊 ANALYSIS FEATURES TEST")
    print("="*60)
    
    commands = [
        ['driftmgr', 'analyze', '--provider', 'aws'],
        ['driftmgr', 'analyze', '--provider', 'azure'],
        ['driftmgr', 'analyze', '--all-providers'],
        ['driftmgr', 'analyze', '--format', 'json']
    ]
    
    for command in commands:
        run_command(command, timeout=60)
        time.sleep(random.uniform(2, 4))

def simulate_state_file_features():
    """Simulate state file features"""
    print("\n" + "="*60)
    print("📁 STATE FILE FEATURES TEST")
    print("="*60)
    
    commands = [
        ['driftmgr', 'statefiles', '--discover'],
        ['driftmgr', 'statefiles', '--analyze'],
        ['driftmgr', 'statefiles', '--validate']
    ]
    
    for command in commands:
        run_command(command, timeout=45)
        time.sleep(random.uniform(1, 2))

def simulate_health_and_monitoring():
    """Simulate health and monitoring features"""
    print("\n" + "="*60)
    print("🏥 HEALTH & MONITORING TEST")
    print("="*60)
    
    commands = [
        ['driftmgr', 'health', '--check'],
        ['driftmgr', 'health', '--status'],
        ['driftmgr', 'server', '--status']
    ]
    
    for command in commands:
        run_command(command, timeout=30)
        time.sleep(random.uniform(1, 2))

def simulate_export_features():
    """Simulate export features"""
    print("\n" + "="*60)
    print("📤 EXPORT FEATURES TEST")
    print("="*60)
    
    commands = [
        ['driftmgr', 'export', '--type', 'resources', '--format', 'json'],
        ['driftmgr', 'export', '--type', 'drift', '--format', 'csv']
    ]
    
    for command in commands:
        run_command(command, timeout=60)
        time.sleep(random.uniform(2, 3))

def simulate_remediation_preview():
    """Simulate remediation preview (dry-run)"""
    print("\n" + "="*60)
    print("🔧 REMEDIATION PREVIEW TEST")
    print("="*60)
    
    commands = [
        ['driftmgr', 'remediate', '--dry-run'],
        ['driftmgr', 'remediate', '--dry-run', '--provider', 'aws']
    ]
    
    for command in commands:
        run_command(command, timeout=60)
        time.sleep(random.uniform(2, 3))

def main():
    """Main simulation function"""
    print("🚀 DriftMgr User Simulation")
    print("="*60)
    print("Simulating user behavior with:")
    print("• Auto-detected credentials")
    print("• Random AWS and Azure regions")
    print("• Core feature testing")
    print("="*60)
    
    # Load regions
    aws_regions, azure_regions = load_regions()
    
    # Check if driftmgr is available
    print("\n🔍 Checking driftmgr availability...")
    if not run_command(['driftmgr', '--version'], timeout=10):
        print("[WARNING] DriftMgr not found or not accessible")
        print("The simulation will run but may show expected failures")
        print("Please ensure driftmgr is installed and in your PATH")
    
    print("\nStarting simulation in 3 seconds...")
    time.sleep(3)
    
    start_time = datetime.now()
    
    # Run simulation phases
    simulate_credential_check()
    simulate_discovery_with_random_regions(aws_regions, azure_regions)
    simulate_analysis_features()
    simulate_state_file_features()
    simulate_health_and_monitoring()
    simulate_export_features()
    simulate_remediation_preview()
    
    end_time = datetime.now()
    duration = end_time - start_time
    
    print("\n" + "="*60)
    print("🎉 SIMULATION COMPLETED")
    print("="*60)
    print(f"Duration: {duration}")
    print(f"AWS regions tested: {min(3, len(aws_regions))}")
    print(f"Azure regions tested: {min(3, len(azure_regions))}")
    print("="*60)
    print("This simulation tested:")
    print("• Credential auto-detection")
    print("• Multi-region discovery")
    print("• Drift analysis")
    print("• State file operations")
    print("• Health monitoring")
    print("• Export capabilities")
    print("• Remediation preview")
    print("="*60)

if __name__ == "__main__":
    main()
