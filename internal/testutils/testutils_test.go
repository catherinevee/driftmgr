package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTestCases(t *testing.T) {
	var testRuns []string

	testCases := []TestCase{
		{
			Name: "test case 1",
			TestFunc: func(t *testing.T) {
				testRuns = append(testRuns, "test1")
			},
		},
		{
			Name: "test case 2",
			TestFunc: func(t *testing.T) {
				testRuns = append(testRuns, "test2")
			},
		},
	}

	RunTestCases(t, testCases)

	assert.Len(t, testRuns, 2)
	assert.Contains(t, testRuns, "test1")
	assert.Contains(t, testRuns, "test2")
}

func TestGetProjectRoot(t *testing.T) {
	root, err := GetProjectRoot()
	require.NoError(t, err)

	// Verify it's the project root by checking for go.mod
	_, err = os.Stat(filepath.Join(root, "go.mod"))
	assert.NoError(t, err)
}

func TestLoadTestFile(t *testing.T) {
	// Create a test file in a known location
	testContent := []byte("test content")
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Test loading the file
	content := LoadTestFile(t, filepath.Join(filepath.Base(tempDir), "testfile.txt"))
	assert.Equal(t, testContent, content)
}

func TestAssertErrorContains(t *testing.T) {
	t.Run("no error expected", func(t *testing.T) {
		AssertErrorContains(t, nil, "")
	})

	t.Run("error with expected message", func(t *testing.T) {
		err := assert.AnError
		AssertErrorContains(t, err, err.Error())
	})
}

func TestCreateTempFile(t *testing.T) {
	content := "test content"
	path := CreateTempFile(t, content)

	// Verify file exists and has the correct content
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestSkipIfShort(t *testing.T) {
	// This is just a compilation test since we can't easily test the skip behavior
	t.Run("not skipped", func(t *testing.T) {
		SkipIfShort(t)
		// Test passes if not skipped
	})
}

func TestSkipIfNotIntegration(t *testing.T) {
	// This is just a compilation test since we can't easily test the skip behavior
	t.Run("not skipped", func(t *testing.T) {
		SkipIfNotIntegration(t)
		// Test passes if not skipped
	})
}
