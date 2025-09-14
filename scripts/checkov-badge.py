#!/usr/bin/env python3
"""
Checkov Security Badge Generator

This script runs Checkov security analysis and generates a dynamic badge
showing the security status of the codebase.
"""

import json
import os
import subprocess
import sys
from pathlib import Path


def run_checkov():
    """Run Checkov security analysis and return results."""
    print(" Running Checkov security analysis...")
    
    try:
        # Run Checkov with JSON output
        result = subprocess.run([
            'checkov', 
            '-d', '.', 
            '--output', 'json',
            '--output-file-path', 'checkov_results.json',
            '--framework', 'dockerfile,kubernetes,secrets,gitlab_ci,github_actions,circleci_pipelines,azure_pipelines,ansible'
        ], capture_output=True, text=True, timeout=300)
        
        print(f"Checkov exit code: {result.returncode}")
        print(f"Checkov stdout: {result.stdout[:200]}...")
        if result.stderr:
            print(f"Checkov stderr: {result.stderr[:200]}...")
        
        return result.returncode, result.stdout, result.stderr
        
    except subprocess.TimeoutExpired:
        print(" Checkov analysis timed out")
        return 1, "", "Timeout"
    except Exception as e:
        print(f" Error running Checkov: {e}")
        return 1, "", str(e)


def parse_checkov_results():
    """Parse Checkov JSON results and calculate security score."""
    try:
        if not os.path.exists('checkov_results.json'):
            print(" Checkov results file not found")
            return 0, "No results", "lightgrey"
        
        with open('checkov_results.json', 'r') as f:
            data = json.load(f)
        
        summary = data.get('summary', {})
        failed_checks = summary.get('failed', 0)
        passed_checks = summary.get('passed', 0)
        total_checks = failed_checks + passed_checks
        
        if total_checks == 0:
            security_score = 100
            color = "brightgreen"
            message = "No checks found"
        else:
            security_score = int((passed_checks / total_checks) * 100)
            
            if security_score >= 90:
                color = "brightgreen"
            elif security_score >= 70:
                color = "yellow"
            else:
                color = "red"
            
            message = f"{security_score}% secure"
        
        print(f" Security Analysis Results:")
        print(f"   Total Checks: {total_checks}")
        print(f"   Passed: {passed_checks}")
        print(f"   Failed: {failed_checks}")
        print(f"   Security Score: {security_score}%")
        print(f"   Badge Color: {color}")
        
        return security_score, message, color
        
    except Exception as e:
        print(f" Error parsing Checkov results: {e}")
        return 0, "Parse error", "lightgrey"


def generate_badge(score, message, color):
    """Generate a dynamic badge using shields.io."""
    print(f" Generating badge: {message} ({color})")
    
    # Create badges directory
    badges_dir = Path("docs/badges")
    badges_dir.mkdir(parents=True, exist_ok=True)
    
    # Generate badge URL
    badge_url = f"https://img.shields.io/badge/Checkov_Security-{message.replace(' ', '%20')}-{color}?style=flat-square&logo=checkov"
    
    # Save badge URL to file
    badge_file = badges_dir / "checkov-security-badge.txt"
    with open(badge_file, 'w') as f:
        f.write(badge_url)
    
    print(f" Badge saved to: {badge_file}")
    print(f" Badge URL: {badge_url}")
    
    return badge_url


def main():
    """Main function to run Checkov analysis and generate badge."""
    print("  Checkov Security Badge Generator")
    print("=" * 50)
    
    # Run Checkov analysis
    exit_code, stdout, stderr = run_checkov()
    
    # Parse results and calculate security score
    score, message, color = parse_checkov_results()
    
    # Generate badge
    badge_url = generate_badge(score, message, color)
    
    print("\n Checkov security badge generation complete!")
    print(f"   Security Score: {score}%")
    print(f"   Badge Message: {message}")
    print(f"   Badge Color: {color}")
    
    # Exit with Checkov's exit code (0 for success, non-zero for issues)
    # But don't fail the workflow if there are security issues
    return 0


if __name__ == "__main__":
    sys.exit(main())
