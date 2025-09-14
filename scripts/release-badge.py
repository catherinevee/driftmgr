#!/usr/bin/env python3
"""
Release Version Badge Generator

This script fetches the latest GitHub release version and generates
a dynamic badge showing the current release version.
"""

import json
import os
import requests
import sys
from pathlib import Path
from datetime import datetime


def get_latest_release():
    """Get the latest release version from GitHub API."""
    print(" Fetching latest release version...")
    
    try:
        # GitHub API endpoint for latest release
        url = "https://api.github.com/repos/catherinevee/driftmgr/releases/latest"
        
        # Make request with proper headers
        headers = {
            'Accept': 'application/vnd.github.v3+json',
            'User-Agent': 'driftmgr-release-badge/1.0'
        }
        
        response = requests.get(url, headers=headers, timeout=30)
        response.raise_for_status()
        
        data = response.json()
        version = data.get('tag_name', 'unknown')
        
        print(f" Latest release version: {version}")
        return version
        
    except requests.exceptions.RequestException as e:
        print(f" Error fetching release version: {e}")
        return "unknown"
    except Exception as e:
        print(f" Unexpected error: {e}")
        return "unknown"


def generate_badge(version):
    """Generate a dynamic badge using shields.io."""
    print(f" Generating badge for version: {version}")
    
    # Remove 'v' prefix if present
    clean_version = version.lstrip('v')
    
    # Generate badge URL using shields.io
    badge_url = f"https://img.shields.io/badge/Release-v{clean_version}-blue?style=flat-square&logo=github&logoColor=white"
    
    print(f" Badge URL: {badge_url}")
    
    return badge_url, clean_version


def save_badge_files(badge_url, version, clean_version):
    """Save badge URL and version info to files."""
    print(" Saving badge files...")
    
    # Create badges directory
    badges_dir = Path("docs/badges")
    badges_dir.mkdir(parents=True, exist_ok=True)
    
    # Save badge URL to text file
    badge_file = badges_dir / "release-version-badge.txt"
    with open(badge_file, 'w') as f:
        f.write(badge_url)
    
    # Save version info to JSON file
    version_info = {
        "version": version,
        "clean_version": clean_version,
        "badge_url": badge_url,
        "last_updated": datetime.utcnow().isoformat() + "Z"
    }
    
    info_file = badges_dir / "release-info.json"
    with open(info_file, 'w') as f:
        json.dump(version_info, f, indent=2)
    
    print(f" Badge saved to: {badge_file}")
    print(f" Version info saved to: {info_file}")
    
    return badge_file, info_file


def main():
    """Main function to generate release badge."""
    print(" DriftMgr Release Version Badge Generator")
    print("=" * 50)
    
    # Get latest release version
    version = get_latest_release()
    
    if version == "unknown":
        print(" Failed to fetch release version")
        return 1
    
    # Generate badge
    badge_url, clean_version = generate_badge(version)
    
    # Save files
    badge_file, info_file = save_badge_files(badge_url, version, clean_version)
    
    print("\n Release version badge generation complete!")
    print(f"   Version: {version}")
    print(f"   Clean Version: {clean_version}")
    print(f"   Badge URL: {badge_url}")
    print(f"   Badge File: {badge_file}")
    print(f"   Info File: {info_file}")
    
    return 0


if __name__ == "__main__":
    sys.exit(main())
