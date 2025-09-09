#!/usr/bin/env python3
"""
Replace all instances of 'comprehensive' with appropriate alternatives in markdown files.
"""

import os
import re

def get_replacement(context):
    """Get appropriate replacement based on context."""
    
    # Define context-specific replacements
    replacements = {
        'comprehensive cloud': 'complete cloud',
        'comprehensive testing': 'thorough testing',
        'comprehensive test': 'complete test',
        'comprehensive guide': 'complete guide',
        'comprehensive documentation': 'detailed documentation',
        'comprehensive coverage': 'full coverage',
        'comprehensive security': 'complete security',
        'comprehensive error': 'detailed error',
        'comprehensive user': 'complete user',
        'comprehensive drift': 'complete drift',
        'comprehensive API': 'complete API',
        'comprehensive strategy': 'complete strategy',
        'comprehensive migration': 'complete migration',
        'comprehensive credential': 'detailed credential',
        'comprehensive examples': 'detailed examples',
        'comprehensive list': 'complete list',
        'comprehensive resource': 'complete resource',
        'comprehensive multi-cloud': 'complete multi-cloud',
        'comprehensive.go': 'comprehensive.go',  # Keep filename as is
    }
    
    # Check for specific phrases
    for phrase, replacement in replacements.items():
        if phrase in context:
            return context.replace(phrase, replacement)
    
    # Default replacement
    return context.replace('comprehensive', 'complete').replace('Comprehensive', 'Complete')

def process_file(filepath):
    """Process a single file to replace 'comprehensive'."""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        
        # Find all instances with context (case-insensitive)
        pattern = re.compile(r'(.{0,30})(comprehensive|Comprehensive)(.{0,30})', re.IGNORECASE)
        
        for match in pattern.finditer(original_content):
            full_match = match.group(0)
            before = match.group(1)
            word = match.group(2)
            after = match.group(3)
            
            # Skip if it's a filename
            if '.go' in after or '.go' in before:
                continue
                
            # Get the full context
            context = before + word + after
            replacement = get_replacement(context)
            
            # Replace in content
            content = content.replace(full_match, replacement)
        
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"[UPDATED] {filepath}")
            return True
        else:
            print(f"[SKIP] {filepath} - No changes needed")
            return False
            
    except Exception as e:
        print(f"[ERROR] {filepath}: {e}")
        return False

def main():
    """Main function to process all markdown files."""
    
    updated_count = 0
    total_count = 0
    
    # Walk through all directories
    for root, dirs, files in os.walk('.'):
        # Skip hidden directories and vendor/node_modules
        dirs[:] = [d for d in dirs if not d.startswith('.') and d not in ['vendor', 'node_modules']]
        
        for file in files:
            if file.endswith('.md'):
                filepath = os.path.join(root, file)
                total_count += 1
                if process_file(filepath):
                    updated_count += 1
    
    # Also check README at root
    if os.path.exists('README.md'):
        total_count += 1
        if process_file('README.md'):
            updated_count += 1
    
    print("-" * 50)
    print(f"Summary: Updated {updated_count} out of {total_count} files")
    print("All instances of 'comprehensive' have been replaced.")

if __name__ == "__main__":
    main()