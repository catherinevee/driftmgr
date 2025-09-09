#!/bin/bash
#
# Install pre-commit hooks for DriftMgr development
#

set -e

echo "Installing pre-commit hooks for DriftMgr..."

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required for pre-commit hooks"
    echo "Please install Python 3 from https://www.python.org/downloads/"
    exit 1
fi

# Check if pip is installed
if ! command -v pip3 &> /dev/null; then
    echo "Error: pip is required"
    echo "Please install pip: python3 -m ensurepip"
    exit 1
fi

# Install pre-commit
echo "Installing pre-commit..."
pip3 install --user pre-commit

# Check if pre-commit is in PATH
if ! command -v pre-commit &> /dev/null; then
    echo "Warning: pre-commit not found in PATH"
    echo "You may need to add Python user scripts to your PATH"
    echo "  Linux/Mac: export PATH=\$HOME/.local/bin:\$PATH"
    echo "  Windows: Add %APPDATA%\\Python\\Scripts to PATH"
fi

# Install the git hook scripts
echo "Installing git hooks..."
pre-commit install

# Install commit message hook
pre-commit install --hook-type commit-msg

# Run against all files to check current status
echo "Running pre-commit checks on existing files..."
pre-commit run --all-files || true

echo ""
echo "✅ Pre-commit hooks installed successfully!"
echo ""
echo "Hooks will now run automatically on git commit."
echo "To run manually: pre-commit run --all-files"
echo "To update hooks: pre-commit autoupdate"
echo ""