// Package testutils provides common utilities for testing across the project.
package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCase represents a test case with a name and test function.
type TestCase struct {
	Name     string
	TestFunc func(t *testing.T)
}

// RunTestCases runs a set of test cases with proper setup and teardown.
func RunTestCases(t *testing.T, testCases []TestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			tc.TestFunc(t)
		})
	}
}

// GetProjectRoot returns the absolute path to the project root.
func GetProjectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get the current file path")
	}

	// Navigate up to the project root (assuming this file is in internal/testutils)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	return projectRoot, nil
}

// LoadTestFile reads and returns the content of a test file.
func LoadTestFile(t *testing.T, relativePath string) []byte {
	projectRoot, err := GetProjectRoot()
	require.NoError(t, err, "Failed to get project root")

	filePath := filepath.Join(projectRoot, relativePath)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err, "Failed to read test file: %s", filePath)

	return content
}

// AssertErrorContains checks if the error message contains the expected text.
func AssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if expected == "" {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), expected)
		}
	}
}

// CreateTempFile creates a temporary file with the given content and returns its path.
// The caller is responsible for cleaning up the file.
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()
	tempFile, err := os.CreateTemp("", "driftmgr-test-*")
	require.NoError(t, err, "Failed to create temp file")

	_, err = tempFile.WriteString(content)
	require.NoError(t, err, "Failed to write to temp file")

	t.Cleanup(func() {
		err := os.Remove(tempFile.Name())
		assert.NoError(t, err, "Failed to cleanup temp file")
	})

	return tempFile.Name()
}

// SkipIfShort skips the test if the -short flag is set.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// SkipIfNotIntegration skips the test if the -integration flag is not set.
func SkipIfNotIntegration(t *testing.T) {
	t.Helper()
	if !isFlagSet("integration") {
		t.Skip("Skipping integration test. Use -integration flag to run.")
	}
}

// isFlagSet checks if the given flag is set in the test flags.
func isFlagSet(name string) bool {
	for _, arg := range os.Args {
		if arg == "-test.run" || arg == "-test.bench" {
			continue
		}
		if arg == "-test.short" && name == "short" {
			return true
		}
		if arg == "-test.v" && name == "v" {
			return true
		}
		if arg == "-"+name || arg == "--"+name {
			return true
		}
	}
	return false
}
