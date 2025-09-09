#!/usr/bin/env python3
"""
Update Go imports after directory restructuring
"""

import os
import re
import sys
from pathlib import Path

# Define import mappings (old -> new)
IMPORT_MAPPINGS = {
    # Core business logic
    '"github.com/catherinevee/driftmgr/internal/discovery"': '"github.com/catherinevee/driftmgr/internal/core/discovery"',
    '"github.com/catherinevee/driftmgr/internal/drift"': '"github.com/catherinevee/driftmgr/internal/core/drift"',
    '"github.com/catherinevee/driftmgr/internal/remediation"': '"github.com/catherinevee/driftmgr/internal/core/remediation"',
    '"github.com/catherinevee/driftmgr/internal/analysis"': '"github.com/catherinevee/driftmgr/internal/core/analysis"',
    '"github.com/catherinevee/driftmgr/internal/models"': '"github.com/catherinevee/driftmgr/internal/core/models"',
    '"github.com/catherinevee/driftmgr/internal/state"': '"github.com/catherinevee/driftmgr/internal/core/state"',
    
    # Cloud providers
    '"github.com/catherinevee/driftmgr/internal/providers/aws"': '"github.com/catherinevee/driftmgr/internal/cloud/aws"',
    '"github.com/catherinevee/driftmgr/internal/providers/azure"': '"github.com/catherinevee/driftmgr/internal/cloud/azure"',
    '"github.com/catherinevee/driftmgr/internal/providers/gcp"': '"github.com/catherinevee/driftmgr/internal/cloud/gcp"',
    '"github.com/catherinevee/driftmgr/internal/providers/digitalocean"': '"github.com/catherinevee/driftmgr/internal/cloud/digitalocean"',
    
    # Infrastructure
    '"github.com/catherinevee/driftmgr/internal/config"': '"github.com/catherinevee/driftmgr/internal/infrastructure/config"',
    '"github.com/catherinevee/driftmgr/internal/cache"': '"github.com/catherinevee/driftmgr/internal/infrastructure/cache"',
    '"github.com/catherinevee/driftmgr/internal/secrets"': '"github.com/catherinevee/driftmgr/internal/infrastructure/secrets"',
    '"github.com/catherinevee/driftmgr/internal/storage"': '"github.com/catherinevee/driftmgr/internal/infrastructure/storage"',
    
    # Security
    '"github.com/catherinevee/driftmgr/internal/security"': '"github.com/catherinevee/driftmgr/internal/security/auth"',
    '"github.com/catherinevee/driftmgr/internal/validation"': '"github.com/catherinevee/driftmgr/internal/security/validation"',
    '"github.com/catherinevee/driftmgr/internal/ratelimit"': '"github.com/catherinevee/driftmgr/internal/security/ratelimit"',
    
    # Observability
    '"github.com/catherinevee/driftmgr/internal/logging"': '"github.com/catherinevee/driftmgr/internal/observability/logging"',
    '"github.com/catherinevee/driftmgr/internal/metrics"': '"github.com/catherinevee/driftmgr/internal/observability/metrics"',
    '"github.com/catherinevee/driftmgr/internal/health"': '"github.com/catherinevee/driftmgr/internal/observability/health"',
    
    # Utils
    '"github.com/catherinevee/driftmgr/internal/errors"': '"github.com/catherinevee/driftmgr/internal/utils/errors"',
    '"github.com/catherinevee/driftmgr/internal/graceful"': '"github.com/catherinevee/driftmgr/internal/utils/graceful"',
    '"github.com/catherinevee/driftmgr/internal/resilience"': '"github.com/catherinevee/driftmgr/internal/utils/circuit"',
    '"github.com/catherinevee/driftmgr/internal/pool"': '"github.com/catherinevee/driftmgr/internal/utils/pool"',
    
    # Integration
    '"github.com/catherinevee/driftmgr/internal/terraform"': '"github.com/catherinevee/driftmgr/internal/integration/terraform"',
    '"github.com/catherinevee/driftmgr/internal/terragrunt"': '"github.com/catherinevee/driftmgr/internal/integration/terragrunt"',
    '"github.com/catherinevee/driftmgr/internal/notification"': '"github.com/catherinevee/driftmgr/internal/integration/notification"',
    
    # API
    '"github.com/catherinevee/driftmgr/internal/api/rest"': '"github.com/catherinevee/driftmgr/internal/app/api"',
    '"github.com/catherinevee/driftmgr/internal/dashboard"': '"github.com/catherinevee/driftmgr/internal/app/dashboard"',
}

def update_imports_in_file(file_path):
    """Update imports in a single Go file"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original = content
        modified = False
        
        # Replace each import
        for old_import, new_import in IMPORT_MAPPINGS.items():
            if old_import in content:
                content = content.replace(old_import, new_import)
                modified = True
                print(f"  Updated: {old_import} -> {new_import}")
        
        # Write back if modified
        if modified:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"[OK] Updated: {file_path}")
            return True
        
        return False
    except Exception as e:
        print(f"[ERROR] Error updating {file_path}: {e}")
        return False

def find_go_files(root_dir):
    """Find all Go files in the project"""
    go_files = []
    for root, dirs, files in os.walk(root_dir):
        # Skip vendor and .git directories
        dirs[:] = [d for d in dirs if d not in ['vendor', '.git', 'node_modules']]
        
        for file in files:
            if file.endswith('.go'):
                go_files.append(os.path.join(root, file))
    
    return go_files

def main():
    """Main function"""
    # Get project root
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    print(f"Project root: {project_root}")
    
    # Find all Go files
    print("\nFinding Go files...")
    go_files = find_go_files(project_root)
    print(f"Found {len(go_files)} Go files")
    
    # Update imports
    print("\nUpdating imports...")
    updated_count = 0
    for file_path in go_files:
        if update_imports_in_file(file_path):
            updated_count += 1
    
    print(f"\n[SUCCESS] Updated {updated_count} files")
    
    # Show summary of changes
    print("\n[SUMMARY] Import Update Summary:")
    print(f"  Total Go files: {len(go_files)}")
    print(f"  Files updated: {updated_count}")
    print(f"  Files unchanged: {len(go_files) - updated_count}")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())