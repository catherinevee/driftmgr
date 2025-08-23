# DriftMgr Gobang-Style TUI

## Overview
DriftMgr now features a Gobang-inspired Terminal User Interface with a three-panel layout similar to database management tools.

## Design Philosophy
Inspired by [Gobang](https://github.com/TaKO8Ki/gobang), a TUI database management tool, this interface provides:
- **Three-panel layout** for efficient navigation
- **Keyboard-driven** with vim-style bindings
- **Dark theme** with clear visual hierarchy
- **Table views** for resource data
- **Real-time filtering** and search

## Layout

```

 DriftMgr - Cloud Resource Manager

 Providers Resources Details

 > AWS Prod Name Type Status Resource Details
 Azure web-server-01 EC2 Running
 GCP database-01 RDS Active
 DO backup-bucket S3 Active ID: i-1234567890
 Name: web-server-01
 Type: EC2 Instance
 Region: us-east-1
 Status: Running
 Cost: $45.20/mo

 Tags:
 Environment: Prod
 Team: Platform

 [Providers] | 4 providers | 245 resources | Ready Tab: Switch | q: Quit

```

## Key Features

### 1. Three-Panel Interface
- **Left Panel**: Cloud providers list
- **Middle Panel**: Resources table with sortable columns
- **Right Panel**: Detailed view of selected resource

### 2. Keyboard Navigation
| Key | Action | Description |
|-----|--------|-------------|
| `Tab` / `Shift+Tab` | Switch panels | Navigate between panels |
| `↑/k` | Move up | Navigate up in lists/tables |
| `↓/j` | Move down | Navigate down in lists/tables |
| `←/h` | Move left | Navigate left |
| `→/l` | Move right | Navigate right |
| `Enter` | Select | Select item and move to next panel |
| `/` or `Ctrl+F` | Filter | Open filter input |
| `r` or `F5` | Refresh | Refresh current data |
| `e` | Export | Export selected resources |
| `d` | Delete | Delete selected resource |
| `g` | Top | Jump to first item |
| `G` | Bottom | Jump to last item |
| `?` | Help | Show help screen |
| `q` | Quit | Exit application |

### 3. Visual Design
- **Tokyo Night** inspired color scheme
- **Rounded borders** for panels
- **Focus indicators** with highlighted borders
- **Status bar** with context information
- **Clean typography** without emojis

### 4. Resource Management
- **Real-time filtering** across all columns
- **Sortable columns** in resource table
- **Detailed resource view** with all metadata
- **Quick actions** (delete, export, refresh)
- **Cost analysis** integrated in views

## Usage Modes

### Default Mode (Gobang-style)
```bash
driftmgr # Launches Gobang-style TUI
```

### Alternative UIs
```bash
driftmgr --modern-tui # Modern Bubble Tea interface
driftmgr --simple-tui # Simple text-based interface
```

### CLI Mode
```bash
driftmgr --help # Show CLI help
driftmgr discover --provider aws
```

## Technical Implementation

### Technologies Used
- **Bubble Tea** - Terminal UI framework
- **Lipgloss** - Styling and layout
- **Bubbles Components**:
 - `list` - Provider selection
 - `table` - Resource display
 - `viewport` - Scrollable details
 - `textinput` - Filter input
 - `help` - Keyboard shortcuts

### Color Palette
```go
Primary: #7aa2f7 // Blue
Secondary: #bb9af7 // Purple
Tertiary: #7dcfff // Cyan
Selected: #9ece6a // Green
Border: #565f89 // Gray
Text: #c0caf5 // Light
Background: #1a1b26 // Dark
```

## Workflow Example

1. **Launch TUI**: Run `driftmgr` without arguments
2. **Select Provider**: Use arrow keys to select AWS/Azure/GCP
3. **View Resources**: Press Enter to load resources in table
4. **Filter Resources**: Press `/` and type to filter
5. **View Details**: Select resource to see full details
6. **Take Action**: Press `d` to delete, `e` to export
7. **Switch Context**: Use Tab to move between panels

## Benefits Over Traditional UI

1. **Efficiency**: All information visible at once
2. **Speed**: Keyboard-only navigation
3. **Clarity**: Clear visual hierarchy
4. **Familiarity**: Similar to popular database tools
5. **Professional**: Clean, modern appearance

## Comparison with Gobang

| Feature | Gobang | DriftMgr TUI |
|---------|--------|--------------|
| Three-panel layout | | |
| Vim keybindings | | |
| Table view | | |
| Real-time filter | | |
| Dark theme | | |
| Status bar | | |
| Purpose | Database management | Cloud resource management |

## Future Enhancements

- [ ] SQL-like query language for resources
- [ ] Multi-tab support for different providers
- [ ] Resource relationship graph view
- [ ] Integrated terminal for cloud CLI commands
- [ ] Customizable color themes
- [ ] Export to multiple formats simultaneously