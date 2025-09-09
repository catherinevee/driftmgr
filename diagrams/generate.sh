#!/bin/bash

# Generate DriftMgr Architecture Diagrams
echo "Generating DriftMgr architecture diagrams..."

# Create output directory
mkdir -p output

# Generate DOT files
echo "Generating architecture diagram..."
go run architecture_diagram.go

echo "Generating drift detection flow diagram..."
go run drift_detection_flow.go

echo "Generating remediation workflow diagram..."
go run remediation_workflow.go

# Convert DOT files to PNG (if Graphviz is available)
echo "Converting DOT files to PNG..."
for dotfile in *.dot; do
    if [ -f "$dotfile" ]; then
        pngfile="${dotfile%.dot}.png"
        if command -v dot >/dev/null 2>&1; then
            dot -Tpng "$dotfile" -o "$pngfile"
            echo "Generated $pngfile"
        else
            echo "Warning: Could not generate $pngfile (Graphviz not available)"
        fi
    fi
done

# Move generated files to output directory
mv *.png output/ 2>/dev/null || true
mv *.svg output/ 2>/dev/null || true
mv *.dot output/ 2>/dev/null || true

echo "Diagrams generated successfully in output/ directory!"
echo "Generated files:"
ls -la output/

echo ""
echo "Note: To generate PNG files from DOT files, install Graphviz:"
echo "  Windows: choco install graphviz"
echo "  macOS:   brew install graphviz"
echo "  Ubuntu:  sudo apt-get install graphviz"