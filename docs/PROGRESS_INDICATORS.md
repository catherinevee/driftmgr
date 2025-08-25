# Progress Indicators in DriftMgr

DriftMgr now includes comprehensive progress indicators to provide better visual feedback during long-running operations.

## Features

### 1. Spinners
Three types of animated spinners for different contexts:

- **Unicode Spinner**: Default spinner with smooth animation using Unicode characters
- **Dot Spinner**: Simple three-dot animation for basic operations  
- **Bar Spinner**: Classic bar rotation animation

### 2. Progress Bars
Visual progress bars with:
- Percentage completion
- Current/total item counts
- ETA calculation for remaining time
- Customizable width

### 3. Loading Animations
Block-based loading animations for indeterminate progress scenarios.

### 4. Multi-Progress Display
Track multiple concurrent operations with individual status indicators.

## Integration Points

Progress indicators have been integrated into the following DriftMgr operations:

### Credential Detection
```
⠋ Detecting cloud credentials
✓ Found 3 configured providers
```

### Resource Discovery
```
Discovering resources [████████████████████░░░░░░░░░░░░░░░░░░░] 55% (27/49) ETA: 12s
```

### Drift Detection
```
⠸ Loading state file
✓ State loaded: 142 resources found

Detecting drift [████████████████████████████████████████] 100% (142/142)
```

### Account Selection
```
⠹ Detecting AWS profiles and accounts
✓ Found 5 AWS profiles
```

## Usage Examples

### In Discovery Operations
When running `driftmgr discover`, you'll see:
1. Loading animation while initializing
2. Progress bar showing provider discovery
3. Individual spinners for each provider being scanned
4. Success/error messages with counts

### In Drift Detection
When running `driftmgr drift detect`, you'll see:
1. Spinner while loading state files
2. Progress bar for resource comparison
3. Status updates for smart defaults application

### In Multi-Provider Operations
When discovering across multiple providers:
```
Discovering providers [██████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 25% (1/4)
⠙ Scanning AWS
✓ Found 127 resources in AWS
⠸ Scanning Azure
✓ Found 89 resources in Azure
```

## Benefits

1. **Better User Experience**: Clear visual feedback during operations
2. **Progress Tracking**: Users know how long operations will take
3. **Operation Status**: Immediate feedback on success or failure
4. **Professional Appearance**: Modern CLI interface

## Technical Details

The progress indicators are implemented in `internal/core/progress/progress.go` and include:

- Thread-safe operations using mutexes
- Configurable update intervals
- Clean terminal output management
- Graceful handling of terminal capabilities

## Customization

Progress indicators automatically adapt to the terminal environment and can be customized through the progress package API:

```go
// Create a custom width progress bar
bar := progress.NewCustomBar(total, width, message)

// Update spinner message dynamically
spinner.UpdateMessage("New status")

// Use different spinner styles
spinner := progress.NewDotSpinner(message)  // Dot style
spinner := progress.NewBarSpinner(message)  // Bar style
```

## Future Enhancements

- Color support for different status states
- Configurable animation speeds
- Terminal width auto-detection
- Progress persistence for resumable operations
- Nested progress indicators for complex workflows