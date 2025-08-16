# DriftMgr Context-Sensitive "?" Help Feature

## Overview

DriftMgr now includes a context-sensitive help system that provides intuitive assistance to users through the familiar "?" command. This feature provides interactive help similar to network devices and command-line tools, making it easy for engineers and DevOps professionals to quickly access help information.

## Features

### 1. Basic "?" Help
- **Command**: `?`
- **Description**: Shows all available commands with descriptions
- **Example**:
  ```
  driftmgr> ?
  Available Commands:
    discover      Discover cloud resources
    analyze       Analyze drift for a state file
    perspective   Compare state with live infrastructure
    visualize     Generate infrastructure visualization
    diagram       Generate infrastructure diagram
    export        Export diagram in specified format
    statefiles    List available state files
    health        Check service health
    notify        Send notifications
    help          Show this help message
    history       Show command history
    clear         Clear the screen
    exit          Exit the shell
    quit          Exit the shell
  ```

### 2. Command-Specific Help
- **Command**: `command ?`
- **Description**: Shows detailed help for a specific command including usage, arguments, and examples
- **Examples**:

#### Discover Command Help
```
driftmgr> discover ?
discover - Discover cloud resources

Usage: discover <provider> [regions...]

Arguments:
  provider    Cloud provider (required)
  regions     Cloud regions (optional, default: us-east-1)

Supported Providers:
  aws         Amazon Web Services
  azure       Microsoft Azure
  gcp         Google Cloud Platform

Examples:
  discover aws
  discover aws us-east-1 us-west-2
  discover azure westeurope
```

#### Analyze Command Help
```
driftmgr> analyze ?
analyze - Analyze drift for a state file

Usage: analyze <statefile_id>

Arguments:
  statefile_id  State file identifier (required)

Examples:
  analyze terraform
  analyze my-project
```

#### Perspective Command Help
```
driftmgr> perspective ?
perspective - Compare state with live infrastructure

Usage: perspective <statefile_id> [provider]

Arguments:
  statefile_id  State file identifier (required)
  provider      Cloud provider (optional, default: aws)

Examples:
  perspective terraform
  perspective terraform aws
  perspective my-project azure
```

#### Notify Command Help
```
driftmgr> notify ?
notify - Send notifications

Usage: notify <type> <subject> <message>

Arguments:
  type      Notification type (required)
  subject   Notification subject (required)
  message   Notification message (required)

Supported Types:
  email     Email notification
  slack     Slack notification
  webhook   Webhook notification

Examples:
  notify email "Drift Alert" "Resources have drifted"
  notify slack "Security Alert" "Unauthorized changes detected"
```

### 3. Partial Command Matching
- **Command**: `partial ?`
- **Description**: Shows suggestions for commands that start with the partial input
- **Example**:
  ```
  driftmgr> disc ?
  No exact match for 'disc'
  
  Did you mean:
    discover
  ```

### 4. Invalid Command Handling
- **Command**: `invalid ?`
- **Description**: Shows helpful error message and suggestions
- **Example**:
  ```
  driftmgr> invalid ?
  No exact match for 'invalid'
  
  Did you mean:
    (no suggestions found)
  
  Type '?' for a list of all available commands
  ```

## Implementation Details

### Architecture
The context-sensitive help system is implemented as part of the interactive shell with the following components:

1. **Context Detection**: Analyzes user input to determine the current command context
2. **Help Functions**: Individual help functions for each command with detailed information
3. **Partial Matching**: Fuzzy matching for incomplete commands
4. **Error Handling**: Graceful handling of invalid commands

### Code Structure
```go
// Main help function
func (shell *InteractiveShell) getContextSensitiveHelp(input string)

// Individual command help functions
func (shell *InteractiveShell) showDiscoverHelp(args []string)
func (shell *InteractiveShell) showAnalyzeHelp(args []string)
func (shell *InteractiveShell) showPerspectiveHelp(args []string)
// ... etc

// Utility functions
func (shell *InteractiveShell) showAllCommands()
func (shell *InteractiveShell) showPartialMatches(partial string)
```

### Integration with Security
The context-sensitive help system integrates seamlessly with the existing security features:

- **Input Validation**: Help requests are processed before validation to ensure they work even with special characters
- **Path Traversal Protection**: Help system doesn't interfere with security measures
- **Command Injection Prevention**: Help is treated as a special case, bypassing normal validation

## Usage Examples

### For New Users
1. Start the interactive shell: `./driftmgr-client.exe`
2. Type `?` to see all available commands
3. Type `discover ?` to learn about the discover command
4. Use the examples provided to execute commands

### For Experienced Users
1. Use `command ?` to quickly check syntax and options
2. Use partial commands with `?` to find the right command
3. Use `?` to verify available options for complex commands

### For Scripting
- The help system works in both interactive and non-interactive modes
- Help output is formatted consistently for parsing
- Error messages are clear and actionable

## Benefits

### 1. Familiar Interface
- Engineers and DevOps professionals already know how to use "?" help
- Reduces learning curve for new users
- Consistent with industry standards

### 2. Context-Sensitive
- Help is relevant to what the user is trying to do
- Shows specific options and examples for each command
- Reduces information overload

### 3. Self-Documenting
- Commands are self-documenting through help
- Examples show real-world usage
- Reduces need for external documentation

### 4. Error Prevention
- Users can verify command syntax before execution
- Clear error messages guide users to correct usage
- Reduces support requests and troubleshooting time

## Testing

The context-sensitive help functionality has been thoroughly tested:

```powershell
# Run the test suite
./test-cisco-help.ps1

# Run the demo
./demo-cisco-help.ps1
```

### Test Coverage
- ✅ Basic "?" help functionality
- ✅ Command-specific help for all commands
- ✅ Partial command matching
- ✅ Invalid command handling
- ✅ Backward compatibility with existing "help" command
- ✅ Security integration
- ✅ Error handling

## Future Enhancements

### Planned Features
1. **Tab Completion**: Auto-complete commands and arguments
2. **Interactive Help**: Step-by-step guided help for complex operations
3. **Help Search**: Search through help content
4. **Help Export**: Export help to various formats (PDF, HTML, etc.)
5. **Custom Help**: Allow users to add custom help content

### Integration Opportunities
1. **API Documentation**: Link help to API documentation
2. **Video Tutorials**: Embed video links in help content
3. **Community Examples**: Include community-contributed examples
4. **Best Practices**: Include best practice recommendations in help

## Conclusion

The context-sensitive "?" help feature significantly improves the user experience of DriftMgr by providing intuitive, interactive assistance. This feature makes the tool more accessible to engineers and DevOps professionals while maintaining the security and functionality of the existing system.

The implementation follows Go best practices and integrates seamlessly with the existing security measures, ensuring that help functionality doesn't compromise the security of the application.
