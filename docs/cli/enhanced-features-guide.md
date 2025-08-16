# DriftMgr CLI Features Guide

## Overview

The DriftMgr CLI includes advanced input features to improve productivity and user experience. This guide covers all the features and how to use them effectively.

## Features

### 1. Tab Completion
- **Command Completion**: Press Tab to complete commands
- **Argument Completion**: Press Tab to complete arguments based on context
- **Provider Completion**: Auto-complete cloud providers (aws, azure, gcp)
- **Region Completion**: Auto-complete regions based on selected provider
- **Resource Completion**: Auto-complete discovered resource names and types
- **State File Completion**: Auto-complete available state file names

### 2. Auto-Suggestions
- **History-Based**: Suggests commands from your command history
- **Command-Based**: Suggests available commands as you type
- **Context-Aware**: Shows relevant suggestions based on current command

### 3. Fuzzy Search
- **Partial Matching**: Find commands and arguments with partial input
- **Case-Insensitive**: Works regardless of case
- **Smart Ranking**: Prioritizes exact matches and prefixes

### 4. Enhanced Navigation
- **Arrow Keys**: Navigate through command history (Up/Down arrows)
- **Cursor Movement**: Left/Right arrows for cursor positioning
- **Backspace**: Proper backspace handling with cursor positioning

## Testing Instructions

### 1. Start the CLI
```bash
./driftmgr-client.exe
```

### 2. Test Tab Completion

#### Command Completion:
```
driftmgr> disc<TAB>
```
Should complete to: `discover`

#### Provider Completion:
```
driftmgr> discover a<TAB>
```
Should complete to: `discover aws`

#### Region Completion:
```
driftmgr> discover aws us<TAB>
```
Should show available US regions

#### Multiple Completions:
```
driftmgr> disc<TAB>
```
Should show: `discover`

### 3. Test Auto-Suggestions

Type partial commands to see suggestions:
```
driftmgr> d
```
Should show suggestions starting with 'd'

### 4. Test Arrow Key Navigation

#### History Navigation:
- Press Up Arrow: Go to previous command
- Press Down Arrow: Go to next command

#### Cursor Movement:
- Press Left Arrow: Move cursor left
- Press Right Arrow: Move cursor right

### 5. Test Fuzzy Search

#### Partial Command Matching:
```
driftmgr> ana<TAB>
```
Should complete to: `analyze`

#### Partial Provider Matching:
```
driftmgr> discover az<TAB>
```
Should complete to: `discover azure`

### 6. Test Context-Aware Completion

#### After Discovery:
1. Run: `discover aws us-east-1`
2. Type: `analyze <TAB>`
3. Should show discovered resource names

#### After State Files:
1. Run: `statefiles`
2. Type: `analyze <TAB>`
3. Should show available state file names

## Expected Behavior

### Tab Completion Examples:

1. **Single Completion**:
   ```
   driftmgr> disc<TAB>
   driftmgr> discover 
   ```

2. **Multiple Completions**:
   ```
   driftmgr> disc<TAB>
   Available completions:
     discover
   driftmgr> discover 
   ```

3. **Context-Aware Completion**:
   ```
   driftmgr> discover a<TAB>
   driftmgr> discover aws 
   ```

4. **Argument Completion**:
   ```
   driftmgr> discover aws us<TAB>
   Available completions:
     us-east-1          us-east-2          us-west-1
     us-west-2
   driftmgr> discover aws us-east-1
   ```

### Auto-Suggestion Examples:

1. **Command Suggestions**:
   ```
   driftmgr> d
   Auto-suggestions:
     discover
     enhanced-discover
   ```

2. **History-Based Suggestions**:
   ```
   driftmgr> disc
   Auto-suggestions:
     discover aws us-east-1
     discover azure westeurope
   ```

## Advanced Features

### 1. Dynamic Completion Updates
- When you discover resources, they become available for completion
- When you list state files, they become available for completion
- Resource names and types are automatically added to completion data

### 2. Smart Argument Completion
- Different commands have different completion contexts
- Provider-specific regions are suggested
- Command-specific options are suggested

### 3. Error Handling
- Graceful handling of invalid input
- Clear error messages for completion failures
- Fallback to basic input when enhanced features fail

## Troubleshooting

### Common Issues:

1. **Tab completion not working**:
   - Ensure you're using the enhanced CLI version
   - Check that the completion.go file is included in the build

2. **Arrow keys not working**:
   - May be terminal-specific issue
   - Try different terminal emulator

3. **Auto-suggestions not showing**:
   - Check that you have command history
   - Ensure input is between 1-10 characters

### Performance Notes:

- Tab completion is instant for small datasets
- Auto-suggestions are limited to 5 results for performance
- History is limited to 100 commands
- Completion data is updated dynamically as you use the CLI

## Future Enhancements

1. **File Path Completion**: Auto-complete file paths for uploads
2. **Advanced Fuzzy Search**: Better ranking algorithms
3. **Custom Completions**: User-defined completion rules
4. **Completion Persistence**: Save completion data between sessions
5. **Multi-line Support**: Enhanced editing for complex commands

## Technical Implementation

### Files Modified:
- `cmd/driftmgr-client/completion.go` - New enhanced input handling
- `cmd/driftmgr-client/main.go` - Integration with enhanced reader
- `README.md` - Updated documentation

### Key Components:
- `EnhancedInputReader` - Main input handling class
- `CompletionData` - Data structure for completion options
- `fuzzySearch()` - Fuzzy search algorithm
- `handleTabCompletion()` - Tab completion logic
- `GetSuggestions()` - Auto-suggestion engine

### Build Command:
```bash
go build -o driftmgr-client.exe cmd/driftmgr-client/main.go cmd/driftmgr-client/completion.go cmd/driftmgr-client/enhanced_analyze.go cmd/driftmgr-client/remediate.go cmd/driftmgr-client/credentials.go
```
