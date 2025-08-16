#!/usr/bin/env python3
"""
TerraVision Integration Script for DriftMgr
Uses TerraVision to generate professional cloud architecture diagrams
"""

import os
import sys
import subprocess
import argparse
import json
from pathlib import Path

def run_command(cmd, cwd=None):
    """Run a command and return the result"""
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
        return result.returncode == 0, result.stdout, result.stderr
    except Exception as e:
        return False, "", str(e)

def generate_terravision_diagram(terraform_dir, output_dir="diagrams", workspace="default"):
    """Generate professional architecture diagram using TerraVision"""
    print(f"Generating TerraVision diagram for: {terraform_dir}")
    
    # Check if terravision is available
    terravision_path = Path("terravision/terravision")
    if not terravision_path.exists():
        print("❌ TerraVision not found. Please ensure it's available in the terravision/ directory.")
        return False
    
    # Create output directory
    os.makedirs(output_dir, exist_ok=True)
    
    # Run TerraVision
    cmd = f"python {terravision_path} draw --source {terraform_dir}"
    if workspace != "default":
        cmd += f" --workspace {workspace}"
    
    success, stdout, stderr = run_command(cmd)
    
    if success:
        print(f"✅ TerraVision diagram generated successfully!")
        print(f"📁 Output: {output_dir}/")
        
        # Move the generated file to output directory if it exists in current dir
        current_diagram = Path("architecture.dot.png")
        if current_diagram.exists():
            output_file = Path(output_dir) / "architecture.png"
            current_diagram.rename(output_file)
            print(f"📄 Diagram saved to: {output_file}")
        
        return True
    else:
        print(f"❌ Failed to generate TerraVision diagram: {stderr}")
        return False

def generate_terravision_json(terraform_dir, output_dir="diagrams", workspace="default"):
    """Generate JSON data using TerraVision"""
    print(f"Generating TerraVision JSON data for: {terraform_dir}")
    
    terravision_path = Path("terravision/terravision")
    if not terravision_path.exists():
        print("❌ TerraVision not found. Please ensure it's available in the terravision/ directory.")
        return False
    
    os.makedirs(output_dir, exist_ok=True)
    
    cmd = f"python {terravision_path} graphdata --source {terraform_dir}"
    if workspace != "default":
        cmd += f" --workspace {workspace}"
    
    success, stdout, stderr = run_command(cmd)
    
    if success:
        output_file = Path(output_dir) / "terravision_data.json"
        with open(output_file, 'w') as f:
            f.write(stdout)
        print(f"✅ TerraVision JSON data saved to: {output_file}")
        return True
    else:
        print(f"❌ Failed to generate TerraVision JSON: {stderr}")
        return False

def create_annotations_file(terraform_dir, annotations_content=None):
    """Create a terravision.yml annotations file"""
    annotations_file = Path(terraform_dir) / "terravision.yml"
    
    if annotations_content is None:
        # Default annotations template
        annotations_content = """format: 0.1
# Main Diagram heading
title: Infrastructure Architecture Diagram

# Add custom annotations here
# connect:
#   aws_instance.web:
#     - aws_elb.main : Load Balancer

# disconnect:
#   aws_cloudwatch_log_group.main:
#     - aws_ecs_service.web

# remove:
#   - aws_iam_role.task_execution_role

# add:
#   aws_subnet.another_one:
#     cidr_block: "10.0.2.0/24"

# update:
#   aws_instance.web:
#     label: "Web Server Instance"
"""
    
    with open(annotations_file, 'w') as f:
        f.write(annotations_content)
    
    print(f"✅ Created annotations file: {annotations_file}")
    return str(annotations_file)

def generate_driftmgr_workflow_diagrams(export_file, terraform_dir, output_dir="diagrams"):
    """Generate before/after diagrams for driftmgr workflow"""
    print("🔄 Generating DriftMgr workflow diagrams...")
    
    os.makedirs(Path(output_dir) / "before-import", exist_ok=True)
    os.makedirs(Path(output_dir) / "after-import", exist_ok=True)
    
    # Generate "before" diagram from export data
    if export_file and os.path.exists(export_file):
        print("📊 Generating 'before' diagram from driftmgr export...")
        # Note: TerraVision works with Terraform files, not export data
        # We'll create a simple representation
        generate_export_summary(export_file, Path(output_dir) / "before-import")
    
    # Generate "after" diagram from Terraform
    if terraform_dir and os.path.exists(terraform_dir):
        print("🏗️ Generating 'after' diagram from Terraform...")
        generate_terravision_diagram(terraform_dir, Path(output_dir) / "after-import")
    
    print("✅ Workflow diagrams generated!")

def generate_export_summary(export_file, output_dir):
    """Generate a summary of driftmgr export data"""
    try:
        with open(export_file, 'r') as f:
            data = json.load(f)
        
        summary_file = Path(output_dir) / "export_summary.md"
        with open(summary_file, 'w') as f:
            f.write("# DriftMgr Export Summary\n\n")
            f.write(f"Total Resources: {len(data)}\n\n")
            
            # Group by resource type
            resource_types = {}
            for resource in data:
                resource_type = resource.get('type', 'unknown')
                if resource_type not in resource_types:
                    resource_types[resource_type] = []
                resource_types[resource_type].append(resource)
            
            f.write("## Resource Breakdown\n\n")
            for resource_type, resources in resource_types.items():
                f.write(f"### {resource_type} ({len(resources)})\n")
                for resource in resources:
                    name = resource.get('name', resource.get('id', 'unknown'))
                    f.write(f"- {name}\n")
                f.write("\n")
        
        print(f"✅ Export summary saved to: {summary_file}")
        
    except Exception as e:
        print(f"❌ Failed to generate export summary: {e}")

def main():
    parser = argparse.ArgumentParser(description="Generate professional infrastructure diagrams using TerraVision")
    parser.add_argument("--terraform-dir", help="Path to Terraform directory")
    parser.add_argument("--export-file", help="Path to driftmgr export file (JSON)")
    parser.add_argument("--output-dir", default="diagrams", help="Output directory for diagrams")
    parser.add_argument("--workspace", default="default", help="Terraform workspace name")
    parser.add_argument("--format", choices=["png", "json", "both"], default="png", 
                       help="Output format")
    parser.add_argument("--create-annotations", action="store_true", 
                       help="Create a terravision.yml annotations file")
    parser.add_argument("--workflow", action="store_true", 
                       help="Generate before/after workflow diagrams")
    
    args = parser.parse_args()
    
    if not args.terraform_dir and not args.export_file:
        print("❌ Please specify either --terraform-dir or --export-file")
        parser.print_help()
        return
    
    # Create output directory
    os.makedirs(args.output_dir, exist_ok=True)
    
    if args.workflow:
        # Generate workflow diagrams
        generate_driftmgr_workflow_diagrams(args.export_file, args.terraform_dir, args.output_dir)
    else:
        if args.terraform_dir:
            # Generate TerraVision diagrams
            if args.format in ["png", "both"]:
                generate_terravision_diagram(args.terraform_dir, args.output_dir, args.workspace)
            
            if args.format in ["json", "both"]:
                generate_terravision_json(args.terraform_dir, args.output_dir, args.workspace)
            
            # Create annotations file if requested
            if args.create_annotations:
                create_annotations_file(args.terraform_dir)
        
        if args.export_file:
            # Generate export summary
            generate_export_summary(args.export_file, args.output_dir)

if __name__ == "__main__":
    main()
