#!/usr/bin/env python3
"""
Remove all emojis from driftmgr codebase
"""

import os
import re
import sys
from pathlib import Path

# Define emoji patterns and their replacements
EMOJI_REPLACEMENTS = {
    '[OK]': '[OK]',
    '[ERROR]': '[ERROR]',
    '[WARNING]': '[WARNING]',
    '🚀': '[DEPLOY]',
    '💡': '[INFO]',
    '🛠️': '[BUILD]',
    '📝': '[DOCS]',
    '🔐': '[SECURITY]',
    '⭐': '[STAR]',
    '🤖': '[AUTO]',
    '📊': '[METRICS]',
    '🔍': '[SEARCH]',
    '🎯': '[TARGET]',
    '🔧': '[FIX]',
    '🏗️': '[BUILD]',
    '🔄': '[SYNC]',
    '[LIGHTNING]': '[PERF]',
    '🔒': '[LOCK]',
    '📈': '[CHART]',
    '💼': '[BUSINESS]',
    '📦': '[PACKAGE]',
    'ℹ️': '[INFO]',
    '👍': '[THUMBSUP]',
    '👎': '[THUMBSDOWN]',
    '🔥': '[HOT]',
    '🐛': '[BUG]',
    '[SPARKLES]': '[NEW]',
    '📚': '[DOCS]',
    '🎨': '[STYLE]',
    '♻️': '[REFACTOR]',
    '🚨': '[ALERT]',
    '🧪': '[TEST]',
    '🔖': '[TAG]',
    '⬆️': '[UPGRADE]',
    '⬇️': '[DOWNGRADE]',
    '📌': '[PIN]',
    '👷': '[CI]',
    '💚': '[CI_FIX]',
    '🔧': '[CONFIG]',
    '🔨': '[SCRIPT]',
    '🌐': '[I18N]',
    '✏️': '[TYPO]',
    '💩': '[POOP]',
    '⏪': '[REVERT]',
    '🔀': '[MERGE]',
    '📦': '[PACKAGE]',
    '👽': '[API]',
    '🚚': '[MOVE]',
    '📄': '[LICENSE]',
    '💥': '[BREAKING]',
    '🍱': '[ASSETS]',
    '♿': '[A11Y]',
    '💄': '[UI]',
    '🔊': '[LOG_ADD]',
    '🔇': '[LOG_REMOVE]',
    '👥': '[CONTRIB]',
    '🚸': '[UX]',
    '🏗': '[ARCH]',
    '📱': '[MOBILE]',
    '🥚': '[EASTER_EGG]',
    '🙈': '[GITIGNORE]',
    '📸': '[SNAPSHOT]',
    '⚗': '[EXPERIMENT]',
    '🔍': '[SEO]',
    '🏷️': '[TYPES]',
    '🌱': '[SEED]',
    '🚩': '[FLAG]',
    '🥅': '[CATCH]',
    '💫': '[ANIMATION]',
    '🗑️': '[CLEANUP]',
    '🛂': '[PASSPORT]',
    '🩹': '[PATCH]',
    '🧐': '[INSPECT]',
    '⚰️': '[DEAD_CODE]',
    '🧑‍💻': '[DEV]',
    '💸': '[MONEY]',
    '🧵': '[THREAD]',
    '🦺': '[SAFETY]',
}

def remove_emojis_from_file(filepath):
    """Remove emojis from a single file"""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        
        # Replace known emojis with text equivalents
        for emoji, replacement in EMOJI_REPLACEMENTS.items():
            if emoji in content:
                content = content.replace(emoji, replacement)
        
        # Remove any remaining unicode emojis (catch-all)
        # This regex matches most emoji ranges
        emoji_pattern = re.compile(
            "["
            u"\U0001F600-\U0001F64F"  # emoticons
            u"\U0001F300-\U0001F5FF"  # symbols & pictographs
            u"\U0001F680-\U0001F6FF"  # transport & map symbols
            u"\U0001F1E0-\U0001F1FF"  # flags (iOS)
            u"\U00002702-\U000027B0"
            u"\U000024C2-\U0001F251"
            u"\U0001F900-\U0001F9FF"  # Supplemental Symbols and Pictographs
            u"\U00002600-\U000026FF"  # Miscellaneous Symbols
            u"\U00002700-\U000027BF"  # Dingbats
            "]+", flags=re.UNICODE
        )
        content = emoji_pattern.sub('', content)
        
        # Only write if content changed
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            return True
    except Exception as e:
        print(f"Error processing {filepath}: {e}")
        return False
    
    return False

def process_directory(root_dir):
    """Process all files in directory"""
    root_path = Path(root_dir)
    extensions = {'.go', '.md', '.yml', '.yaml', '.sh', '.ps1', '.py', '.txt', '.json', '.tf', '.bat'}
    
    files_processed = 0
    files_modified = 0
    
    for filepath in root_path.rglob('*'):
        # Skip .git directory and node_modules
        if '.git' in filepath.parts or 'node_modules' in filepath.parts:
            continue
            
        if filepath.is_file() and filepath.suffix in extensions:
            files_processed += 1
            if remove_emojis_from_file(filepath):
                files_modified += 1
                print(f"Modified: {filepath}")
    
    print(f"\nProcessed {files_processed} files")
    print(f"Modified {files_modified} files")

def main():
    # Get driftmgr root directory
    if len(sys.argv) > 1:
        root_dir = sys.argv[1]
    else:
        root_dir = r"C:\Users\cathe\OneDrive\Desktop\github\driftmgr"
    
    if not os.path.exists(root_dir):
        print(f"Error: Directory {root_dir} does not exist")
        sys.exit(1)
    
    print(f"Removing emojis from files in: {root_dir}")
    process_directory(root_dir)
    print("Done!")

if __name__ == "__main__":
    main()