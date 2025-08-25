# Color Support in DriftMgr

DriftMgr now includes comprehensive color support to improve visual clarity and help users quickly identify important information.

## Features

### Automatic Detection
- Colors are automatically enabled when running in a terminal that supports ANSI colors
- Windows 10+ with virtual terminal processing is supported
- Colors are disabled when output is piped or redirected
- Respects `NO_COLOR` environment variable
- Can be forced with `FORCE_COLOR` environment variable

### Color Categories

#### Provider-Specific Colors
Each cloud provider has its own distinctive color:
- **AWS**: Yellow/Orange
- **Azure**: Blue  
- **GCP**: Bright Blue
- **DigitalOcean**: Cyan

#### Status Indicators
Visual feedback for different states:
- ✓ **Success**: Green
- ✗ **Error**: Red
- ⚠ **Warning**: Yellow
- → **Info**: Cyan
- • **Neutral**: Gray

#### Severity Levels
Drift severity is color-coded:
- **Critical**: Bright Red - Immediate action required
- **High**: Red - Significant issues
- **Medium**: Yellow - Moderate priority
- **Low**: Blue - Minor issues

#### UI Elements
Different colors for various UI components:
- **Headers**: Bold white
- **Subheaders**: Bold cyan
- **Labels**: Bold white
- **Values**: Bright white
- **Commands**: Bright green
- **Flags**: Yellow
- **Paths**: Bright blue
- **Dimmed text**: Gray

### Resource Count Coloring
Resource counts are automatically colored based on quantity:
- **0**: Gray (no resources)
- **1-100**: Green (normal)
- **101-500**: Yellow (elevated)
- **500+**: Red (high)

## Usage Examples

### Credential Display
```
─────────────────────────────────────────────
AWS:            ✓ Configured
                AWS Account 123456789012
                Available profiles: default, production
Azure:          ✓ Configured  
                Subscription: Production
GCP:            ✗ Not configured
─────────────────────────────────────────────
```

### Discovery Output
```
=== Discovery Summary ===
Total Resources: 342

By Provider:
  AWS: 215
  Azure: 89
  GCP: 38
```

### Drift Detection
```
Smart defaults: Filtered 45 harmless drift items (72.5% noise reduction)

[WARNING] 3 CRITICAL drift items detected requiring immediate attention!
```

### Help Text
Commands are shown in green, flags in yellow, and descriptions in gray for easy scanning.

## Environment Variables

### Disable Colors
```bash
NO_COLOR=1 driftmgr status
```

### Force Colors (even when piped)
```bash
FORCE_COLOR=1 driftmgr discover | tee output.log
```

## Benefits

1. **Improved Readability**: Important information stands out
2. **Quick Scanning**: Color coding allows faster identification of issues
3. **Provider Recognition**: Each provider has a unique color identity
4. **Severity Assessment**: Instantly see critical vs minor issues
5. **Professional Appearance**: Modern, polished CLI experience

## Implementation Details

The color system is implemented in `internal/core/color/color.go` and includes:

- ANSI escape code management
- Terminal capability detection
- Windows compatibility
- Color stripping for non-terminal output
- Semantic color functions
- Provider-specific theming

## Accessibility

- Colors supplement but don't replace text information
- All colored elements include text labels
- NO_COLOR environment variable support for accessibility tools
- High contrast color choices for visibility

## Future Enhancements

- User-configurable color themes
- 256-color and true color support
- Background color support for alerts
- Animated color transitions
- Color profiles for different terminal themes