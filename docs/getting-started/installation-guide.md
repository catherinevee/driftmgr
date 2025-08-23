# DriftMgr Installation Guide

## Quick Install (Windows)

### Method 1: Automated Installation (Recommended)
```batch
# Run the installation script
install.bat
```

This will:
- Copy `driftmgr.exe` to `%LOCALAPPDATA%\DriftMgr`
- Add the directory to your system PATH
- Make `driftmgr` available from any terminal

**Note:** You must restart your terminal after installation for PATH changes to take effect.

### Method 2: Manual Installation

1. **Copy the executable:**
 ```batch
 mkdir %LOCALAPPDATA%\DriftMgr
 copy driftmgr.exe %LOCALAPPDATA%\DriftMgr\
 ```

2. **Add to PATH:**
 - Open System Properties → Advanced → Environment Variables
 - Under "User variables", select "Path" and click "Edit"
 - Click "New" and add: `%LOCALAPPDATA%\DriftMgr`
 - Click "OK" to save

3. **Restart your terminal**

## Verify Installation

Open a new terminal and run:
```batch
driftmgr --help
```

## Usage

Once installed, you can run DriftMgr from anywhere:

```batch
# Launch interactive TUI
driftmgr

# Launch enhanced TUI
driftmgr --enhanced

# Show help
driftmgr --help

# Discover cloud resources
driftmgr discover

# Detect drift
driftmgr drift detect
```

## Temporary Usage (Current Session Only)

If you want to use DriftMgr immediately without restarting your terminal:

```batch
# Add to current session PATH
add_to_path.bat

# Now you can use driftmgr
driftmgr
```

## Building from Source

If you need to rebuild the executable:

```batch
# Build the executable
go build -o driftmgr.exe ./cmd/driftmgr

# Then run installation
install.bat
```

## Uninstallation

To remove DriftMgr:

1. Delete the installation directory:
 ```batch
 rmdir /s %LOCALAPPDATA%\DriftMgr
 ```

2. Remove from PATH:
 - Open System Properties → Advanced → Environment Variables
 - Edit the Path variable and remove the DriftMgr entry

## Alternative Locations

You can also install DriftMgr to other locations in your PATH:

- `C:\Windows\System32` (requires admin, not recommended)
- `C:\Program Files\DriftMgr` (requires admin)
- Any directory already in your PATH

## Troubleshooting

### "driftmgr is not recognized as an internal or external command"

This means the PATH hasn't been updated. Try:
1. Restart your terminal (CMD/PowerShell)
2. Run `add_to_path.bat` for current session
3. Verify PATH: `echo %PATH%` should contain `%LOCALAPPDATA%\DriftMgr`

### Permission Denied

Run the installation as your normal user account (not as Administrator unless installing to system directories).

### PowerShell Execution Policy

If using PowerShell scripts, you may need to bypass execution policy:
```powershell
powershell -ExecutionPolicy Bypass -File install_windows.ps1
```

## Linux/macOS Installation

For Unix-like systems:

```bash
# Build
go build -o driftmgr ./cmd/driftmgr

# Install to local bin
mkdir -p ~/.local/bin
cp driftmgr ~/.local/bin/
chmod +x ~/.local/bin/driftmgr

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"

# Or install system-wide
sudo cp driftmgr /usr/local/bin/
```

## Docker Installation

You can also run DriftMgr in a container:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o driftmgr ./cmd/driftmgr

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/driftmgr /usr/local/bin/
ENTRYPOINT ["driftmgr"]
```

## Cloud Shell Installation

For Google Cloud Shell, AWS CloudShell, or Azure Cloud Shell:

```bash
# Download and install
curl -LO https://github.com/yourusername/driftmgr/releases/latest/download/driftmgr
chmod +x driftmgr
sudo mv driftmgr /usr/local/bin/
```