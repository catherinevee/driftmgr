# DriftMgr TUI Demo

## Overview
DriftMgr now includes an integrated Terminal User Interface (TUI) that launches automatically when you run `driftmgr.exe` without any arguments.

## Features

### Modern TUI (Default)
When you run `driftmgr.exe` or `driftmgr.exe --tui`, you get:

```
╭─────────────────────────────────────────────────────────────────╮
│  DriftMgr - Cloud Resource Discovery Tool                      │
├─────────────────────────────────────────────────────────────────┤
│  Discover, analyze, and manage your cloud infrastructure       │
│                                                                 │
│  ◎ Discover Resources                                          │
│     Scan cloud providers for resources                         │
│                                                                 │
│  ▣ List Accounts                                               │
│     View configured cloud accounts                             │
│                                                                 │
│  ⬇ Export Results                                              │
│     Export discovery results to various formats                │
│                                                                 │
│  ⚙ Configuration                                               │
│     Manage settings and credentials                            │
│                                                                 │
│  ? Help                                                         │
│     View documentation and shortcuts                           │
╰─────────────────────────────────────────────────────────────────╯
 DriftMgr v1.0.0                        Press ? for help | q to quit
```

### Navigation
- **↑/k** - Move up
- **↓/j** - Move down  
- **Enter** - Select item
- **Esc/b** - Go back
- **q** - Quit
- **?** - Show help

### Discovery View
```
╭─────────────────────────────────────────────────────────────────╮
│  Resource Discovery Configuration                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Cloud Provider:                                                │
│    ○ AWS                                                        │
│    ● Azure                                                      │
│    ○ GCP                                                        │
│    ○ DigitalOcean                                              │
│                                                                 │
│  Regions:                                                       │
│    ● All Regions                                                │
│    ○ us-east-1                                                  │
│    ○ us-west-2                                                  │
│    ○ eu-west-1                                                  │
│                                                                 │
│  Options:                                                       │
│    ☑ Enable cost analysis                                      │
│    ☐ Include terminated resources                              │
│    ☑ Parallel discovery                                        │
│    ☐ Deep scan mode                                            │
│                                                                 │
│  Use arrow keys to navigate, space to select, enter to start   │
╰─────────────────────────────────────────────────────────────────╯
```

### Simple TUI
Run `driftmgr.exe --simple-tui` for a text-only interface:

```
══════════════════════════════════════════════════════════════════════
  DriftMgr - Cloud Resource Discovery & Analysis
══════════════════════════════════════════════════════════════════════

  Discover, analyze, and manage cloud resources across providers
  with cost analysis and comprehensive export capabilities

  MAIN MENU
  ──────────────────────────────────────────────────────────────────

  [1] Discover Cloud Resources
  [2] Resource Discovery with Cost Analysis
  [3] Export Discovery Results
  [4] List Cloud Accounts
  [5] Configuration & Settings
  [6] Help & Documentation
  [7] Exit

  ──────────────────────────────────────────────────────────────────
  Select option: _
```

## Usage Modes

### 1. Interactive TUI (No Arguments)
```bash
driftmgr.exe
```
Launches the modern Bubble Tea TUI with full mouse and keyboard support.

### 2. Explicit TUI Mode
```bash
driftmgr.exe --tui      # Modern TUI
driftmgr.exe --simple-tui  # Simple text-based TUI
```

### 3. CLI Mode (With Arguments)
```bash
driftmgr.exe --help
driftmgr.exe discover --provider aws --regions us-east-1
driftmgr.exe delete-resource eks_cluster my-cluster
```

## Benefits

1. **User-Friendly**: No need to remember command syntax
2. **Visual Feedback**: Progress bars, spinners, and status indicators
3. **Discoverable**: All features accessible through menus
4. **Efficient**: Keyboard shortcuts for power users
5. **Flexible**: Switch between TUI and CLI as needed
6. **Professional**: Clean, modern interface similar to k9s, lazygit, etc.

## Technical Details

- Built with **Bubble Tea** framework (same as GitHub CLI)
- **Lipgloss** for styling and colors
- Responsive design adapts to terminal size
- Full keyboard navigation with vim-style bindings
- No external dependencies beyond Go standard library