package osutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMac returns true if running on macOS
func IsMac() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// GetHomeDir returns the user's home directory
func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variables
		if IsWindows() {
			return os.Getenv("USERPROFILE")
		}
		return os.Getenv("HOME")
	}
	return home
}

// GetConfigDir returns the appropriate config directory for the OS
func GetConfigDir(appName string) string {
	home := GetHomeDir()

	if IsWindows() {
		// Windows: %APPDATA%\appname
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, appName)
		}
		return filepath.Join(home, "AppData", "Roaming", appName)
	}

	// Unix-like: ~/.config/appname
	return filepath.Join(home, ".config", appName)
}

// GetCacheDir returns the appropriate cache directory for the OS
func GetCacheDir(appName string) string {
	home := GetHomeDir()

	if IsWindows() {
		// Windows: %LOCALAPPDATA%\appname\cache
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, appName, "cache")
		}
		return filepath.Join(home, "AppData", "Local", appName, "cache")
	}

	if IsMac() {
		// macOS: ~/Library/Caches/appname
		return filepath.Join(home, "Library", "Caches", appName)
	}

	// Linux: ~/.cache/appname
	return filepath.Join(home, ".cache", appName)
}

// ExecCommand executes a command with proper OS handling
func ExecCommand(name string, args ...string) ([]byte, error) {
	var cmd *exec.Cmd

	if IsWindows() {
		// Check if command needs to be run through cmd.exe
		if needsCmdWrapper(name) {
			cmdArgs := append([]string{"/c", name}, args...)
			cmd = exec.Command("cmd.exe", cmdArgs...)
		} else {
			cmd = exec.Command(name, args...)
		}
	} else {
		cmd = exec.Command(name, args...)
	}

	return cmd.Output()
}

// ExecCommandWithEnv executes a command with environment variables
func ExecCommandWithEnv(env []string, name string, args ...string) ([]byte, error) {
	var cmd *exec.Cmd

	if IsWindows() && needsCmdWrapper(name) {
		cmdArgs := append([]string{"/c", name}, args...)
		cmd = exec.Command("cmd.exe", cmdArgs...)
	} else {
		cmd = exec.Command(name, args...)
	}

	cmd.Env = append(os.Environ(), env...)
	return cmd.Output()
}

// needsCmdWrapper checks if a command needs cmd.exe wrapper on Windows
func needsCmdWrapper(command string) bool {
	// Commands that typically need cmd.exe wrapper
	needsWrapper := []string{
		"az", "aws", "gcloud", "doctl", "terraform",
		"kubectl", "helm", "docker", "npm", "pip",
	}

	cmdLower := strings.ToLower(command)
	for _, cmd := range needsWrapper {
		if cmdLower == cmd {
			return true
		}
	}

	return false
}

// FindExecutable finds an executable in PATH or common locations
func FindExecutable(name string) (string, error) {
	// First try PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	// Try common locations based on OS
	var locations []string

	if IsWindows() {
		locations = []string{
			filepath.Join(os.Getenv("ProgramFiles"), name),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), name),
			filepath.Join(os.Getenv("LOCALAPPDATA"), name),
			filepath.Join(GetHomeDir(), "AppData", "Local", name),
			filepath.Join(GetHomeDir(), name),
		}

		// Add .exe extension if not present
		if !strings.HasSuffix(name, ".exe") {
			name = name + ".exe"
		}
	} else {
		locations = []string{
			filepath.Join("/usr/local/bin", name),
			filepath.Join("/usr/bin", name),
			filepath.Join("/opt", name, "bin", name),
			filepath.Join(GetHomeDir(), ".local", "bin", name),
			filepath.Join(GetHomeDir(), "bin", name),
		}
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}

		// Try with executable name in subdirectory
		execPath := filepath.Join(loc, name)
		if _, err := os.Stat(execPath); err == nil {
			return execPath, nil
		}
	}

	return "", exec.ErrNotFound
}

// GetAWSConfigPath returns the AWS config directory path
func GetAWSConfigPath() string {
	if awsConfig := os.Getenv("AWS_CONFIG_FILE"); awsConfig != "" {
		return filepath.Dir(awsConfig)
	}
	return filepath.Join(GetHomeDir(), ".aws")
}

// GetAzureConfigPath returns the Azure config directory path
func GetAzureConfigPath() string {
	if IsWindows() {
		return filepath.Join(GetHomeDir(), ".azure")
	}
	return filepath.Join(GetHomeDir(), ".azure")
}

// GetGCPConfigPath returns the GCP config directory path
func GetGCPConfigPath() string {
	return filepath.Join(GetHomeDir(), ".config", "gcloud")
}

// GetDOConfigPath returns the DigitalOcean config directory path
func GetDOConfigPath() string {
	if IsWindows() {
		return filepath.Join(GetHomeDir(), "AppData", "Roaming", "doctl")
	}
	return filepath.Join(GetHomeDir(), ".config", "doctl")
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// PathSeparator returns the OS-specific path separator
func PathSeparator() string {
	return string(os.PathSeparator)
}

// JoinPath joins path elements with OS-specific separator
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}
