#!/usr/bin/env python3
"""
Terraform Graph Generation Script for DriftMgr
Integrates multiple visualization tools for creating infrastructure diagrams
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

def generate_terraform_graph(terraform_dir, output_file="terraform-graph.dot"):
    """Generate Terraform dependency graph using terraform graph command"""
    print(f"Generating Terraform graph for: {terraform_dir}")
    
    success, stdout, stderr = run_command("terraform graph", cwd=terraform_dir)
    
    if success:
        with open(output_file, 'w') as f:
            f.write(stdout)
        print(f"✅ Terraform graph saved to: {output_file}")
        return True
    else:
        print(f"❌ Failed to generate Terraform graph: {stderr}")
        return False

def generate_blast_radius_graph(terraform_dir, output_dir="diagrams", format="html"):
    """Generate Blast Radius visualization"""
    print(f"Generating Blast Radius diagram for: {terraform_dir}")
    
    # Check if blast_radius.py exists
    blast_radius_path = Path("../blast-radius/blast_radius.py")
    if not blast_radius_path.exists():
        print("❌ blast_radius.py not found. Please ensure it's available.")
        return False
    
    cmd = f"python {blast_radius_path} --export {terraform_dir} --format {format} --output {output_dir}"
    success, stdout, stderr = run_command(cmd)
    
    if success:
        print(f"✅ Blast Radius diagram saved to: {output_dir}")
        return True
    else:
        print(f"❌ Failed to generate Blast Radius diagram: {stderr}")
        return False

def serve_blast_radius(terraform_dir, host="localhost", port=5000):
    """Serve Blast Radius web interface"""
    print(f"Starting Blast Radius web server for: {terraform_dir}")
    
    blast_radius_path = Path("../blast-radius/blast_radius.py")
    if not blast_radius_path.exists():
        print("❌ blast_radius.py not found. Please ensure it's available.")
        return False
    
    cmd = f"python {blast_radius_path} --serve {terraform_dir} --host {host} --port {port}"
    print(f"🌐 Starting web server at: http://{host}:{port}")
    print("Press Ctrl+C to stop the server")
    
    try:
        subprocess.run(cmd, shell=True)
    except KeyboardInterrupt:
        print("\n🛑 Server stopped")
        return True

def convert_dot_to_image(dot_file, output_format="png"):
    """Convert DOT file to image using GraphViz"""
    print(f"Converting {dot_file} to {output_format}")
    
    # Check if dot command is available
    success, _, _ = run_command("dot -V")
    if not success:
        print("❌ GraphViz 'dot' command not found. Please install GraphViz.")
        print("   Download from: https://graphviz.org/download/")
        return False
    
    output_file = dot_file.replace('.dot', f'.{output_format}')
    cmd = f"dot -T{output_format} {dot_file} -o {output_file}"
    
    success, stdout, stderr = run_command(cmd)
    if success:
        print(f"✅ Image saved to: {output_file}")
        return True
    else:
        print(f"❌ Failed to convert DOT file: {stderr}")
        return False

def generate_driftmgr_export_graph(export_file, output_dir="diagrams"):
    """Generate graph from driftmgr export data"""
    print(f"Generating graph from driftmgr export: {export_file}")
    
    if not os.path.exists(export_file):
        print(f"❌ Export file not found: {export_file}")
        return False
    
    try:
        with open(export_file, 'r') as f:
            data = json.load(f)
        
        # Create a simple DOT graph from the export data
        dot_content = "digraph DriftMgrExport {\n"
        dot_content += "  rankdir=TB;\n"
        dot_content += "  node [shape=box, style=filled, fillcolor=lightblue];\n"
        
        for resource in data:
            resource_id = resource.get('id', resource.get('name', 'unknown'))
            resource_type = resource.get('type', 'unknown')
            dot_content += f'  "{resource_id}" [label="{resource_type}\\n{resource_id}"];\n'
        
        dot_content += "}\n"
        
        output_file = os.path.join(output_dir, "driftmgr-export.dot")
        os.makedirs(output_dir, exist_ok=True)
        
        with open(output_file, 'w') as f:
            f.write(dot_content)
        
        print(f"✅ DriftMgr export graph saved to: {output_file}")
        return True
        
    except Exception as e:
        print(f"❌ Failed to generate export graph: {e}")
        return False

def main():
    parser = argparse.ArgumentParser(description="Generate Terraform infrastructure diagrams")
    parser.add_argument("--terraform-dir", help="Path to Terraform directory")
    parser.add_argument("--export-file", help="Path to driftmgr export file (JSON)")
    parser.add_argument("--output-dir", default="diagrams", help="Output directory for diagrams")
    parser.add_argument("--format", choices=["dot", "png", "svg", "html", "all"], default="dot", 
                       help="Output format")
    parser.add_argument("--serve", action="store_true", help="Start web server for interactive visualization")
    parser.add_argument("--host", default="localhost", help="Web server host")
    parser.add_argument("--port", type=int, default=5000, help="Web server port")
    
    args = parser.parse_args()
    
    # Create output directory
    os.makedirs(args.output_dir, exist_ok=True)
    
    if args.terraform_dir:
        if args.serve:
            serve_blast_radius(args.terraform_dir, args.host, args.port)
        else:
            # Generate Terraform graph
            dot_file = os.path.join(args.output_dir, "terraform-graph.dot")
            if generate_terraform_graph(args.terraform_dir, dot_file):
                if args.format in ["png", "svg"]:
                    convert_dot_to_image(dot_file, args.format)
                elif args.format == "all":
                    convert_dot_to_image(dot_file, "png")
                    convert_dot_to_image(dot_file, "svg")
            
            # Generate Blast Radius diagram
            if args.format in ["html", "all"]:
                generate_blast_radius_graph(args.terraform_dir, args.output_dir, "html")
    
    if args.export_file:
        generate_driftmgr_export_graph(args.export_file, args.output_dir)
    
    if not args.terraform_dir and not args.export_file:
        print("❌ Please specify either --terraform-dir or --export-file")
        parser.print_help()

if __name__ == "__main__":
    main()
