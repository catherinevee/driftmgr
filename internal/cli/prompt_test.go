package cli

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPrompt tests the prompt functionality
func TestPrompt(t *testing.T) {
	t.Run("NewPrompt", func(t *testing.T) {
		prompt := NewPrompt()

		assert.NotNil(t, prompt)
		assert.NotNil(t, prompt.reader)
		assert.NotNil(t, prompt.formatter)
	})

	t.Run("Confirm_yes_input", func(t *testing.T) {
		input := "y\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result := prompt.Confirm("Are you sure?", false)
		assert.True(t, result)
	})

	t.Run("Confirm_no_input", func(t *testing.T) {
		input := "n\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result := prompt.Confirm("Are you sure?", true)
		assert.False(t, result)
	})

	t.Run("Confirm_empty_input_default_yes", func(t *testing.T) {
		input := "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result := prompt.Confirm("Are you sure?", true)
		assert.True(t, result)
	})

	t.Run("Confirm_empty_input_default_no", func(t *testing.T) {
		input := "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result := prompt.Confirm("Are you sure?", false)
		assert.False(t, result)
	})

	t.Run("Confirm_yes_variations", func(t *testing.T) {
		yesInputs := []string{"y\n", "Y\n", "yes\n", "YES\n", "Yes\n"}

		for _, input := range yesInputs {
			prompt := &Prompt{
				reader:    bufio.NewReader(strings.NewReader(input)),
				formatter: NewOutputFormatter(),
			}

			result := prompt.Confirm("Are you sure?", false)
			assert.True(t, result, "Input '%s' should return true", input)
		}
	})

	t.Run("Confirm_no_variations", func(t *testing.T) {
		noInputs := []string{"n\n", "N\n", "no\n", "NO\n", "No\n"}

		for _, input := range noInputs {
			prompt := &Prompt{
				reader:    bufio.NewReader(strings.NewReader(input)),
				formatter: NewOutputFormatter(),
			}

			result := prompt.Confirm("Are you sure?", true)
			assert.False(t, result, "Input '%s' should return false", input)
		}
	})

	t.Run("Confirm_invalid_input_default_yes", func(t *testing.T) {
		input := "maybe\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result := prompt.Confirm("Are you sure?", true)
		assert.False(t, result)
	})

	t.Run("Confirm_invalid_input_default_no", func(t *testing.T) {
		input := "maybe\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result := prompt.Confirm("Are you sure?", false)
		assert.False(t, result)
	})
}

// TestPromptWithDetails tests the ConfirmWithDetails functionality
func TestPromptWithDetails(t *testing.T) {
	t.Run("ConfirmWithDetails_yes", func(t *testing.T) {
		input := "y\n"
		var buf bytes.Buffer
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}
		prompt.formatter.writer = &buf

		details := []string{"Detail 1", "Detail 2", "Detail 3"}
		result := prompt.ConfirmWithDetails("Proceed with operation?", details)

		assert.True(t, result)
		assert.Contains(t, buf.String(), "Proceed with operation?")
	})

	t.Run("ConfirmWithDetails_no", func(t *testing.T) {
		input := "n\n"
		var buf bytes.Buffer
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}
		prompt.formatter.writer = &buf

		details := []string{"Detail 1", "Detail 2"}
		result := prompt.ConfirmWithDetails("Proceed with operation?", details)

		assert.False(t, result)
	})

	t.Run("ConfirmWithDetails_empty_details", func(t *testing.T) {
		input := "y\n"
		var buf bytes.Buffer
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}
		prompt.formatter.writer = &buf

		result := prompt.ConfirmWithDetails("Proceed?", []string{})

		assert.True(t, result)
	})

	t.Run("ConfirmWithDetails_nil_details", func(t *testing.T) {
		input := "y\n"
		var buf bytes.Buffer
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}
		prompt.formatter.writer = &buf

		result := prompt.ConfirmWithDetails("Proceed?", nil)

		assert.True(t, result)
	})
}

// TestPromptInput tests the Input functionality
func TestPromptInput(t *testing.T) {
	t.Run("Input_basic", func(t *testing.T) {
		input := "test input\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "")
		assert.NoError(t, err)
		assert.Equal(t, "test input", result)
	})

	t.Run("Input_with_default", func(t *testing.T) {
		input := "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "default value")
		assert.NoError(t, err)
		assert.Equal(t, "default value", result)
	})

	t.Run("Input_with_custom_value", func(t *testing.T) {
		input := "custom value\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "default value")
		assert.NoError(t, err)
		assert.Equal(t, "custom value", result)
	})

	t.Run("Input_trimmed", func(t *testing.T) {
		input := "  test input  \n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "")
		assert.NoError(t, err)
		assert.Equal(t, "test input", result)
	})
}

// TestPromptSelect tests the Select functionality
func TestPromptSelect(t *testing.T) {
	t.Run("Select_valid_choice", func(t *testing.T) {
		input := "2\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2", "Option 3"}
		result, err := prompt.Select("Choose an option:", options)

		assert.NoError(t, err)
		assert.Equal(t, 1, result) // 0-indexed
	})

	t.Run("Select_invalid_choice_default", func(t *testing.T) {
		input := "5\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2", "Option 3"}
		result, err := prompt.Select("Choose an option:", options)

		assert.Error(t, err)        // Should return error for invalid choice
		assert.Equal(t, -1, result) // Should return -1 on error
	})

	t.Run("Select_empty_input", func(t *testing.T) {
		input := "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2"}
		result, err := prompt.Select("Choose an option:", options)

		assert.Error(t, err)        // Empty input should return error
		assert.Equal(t, -1, result) // Should return -1 on error
	})

	t.Run("Select_single_option", func(t *testing.T) {
		input := "1\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Only Option"}
		result, err := prompt.Select("Choose an option:", options)

		assert.NoError(t, err)
		assert.Equal(t, 0, result)
	})

	t.Run("Select_empty_options", func(t *testing.T) {
		input := "1\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{}
		result, err := prompt.Select("Choose an option:", options)

		assert.Error(t, err)        // Should return error for empty options
		assert.Equal(t, -1, result) // Should return -1 for empty options
	})
}

// TestPromptMultiSelect tests the MultiSelect functionality
func TestPromptMultiSelect(t *testing.T) {
	t.Run("MultiSelect_single_choice", func(t *testing.T) {
		input := "1\n\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2", "Option 3"}
		result, err := prompt.MultiSelect("Choose options:", options)

		assert.NoError(t, err)
		assert.Equal(t, []int{0}, result)
	})

	t.Run("MultiSelect_multiple_choices", func(t *testing.T) {
		input := "1,2\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2", "Option 3"}
		result, err := prompt.MultiSelect("Choose options:", options)

		assert.NoError(t, err)
		assert.Equal(t, []int{0, 1}, result)
	})

	t.Run("MultiSelect_empty_input", func(t *testing.T) {
		input := "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2"}
		result, err := prompt.MultiSelect("Choose options:", options)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("MultiSelect_invalid_choice", func(t *testing.T) {
		input := "5\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		options := []string{"Option 1", "Option 2"}
		result, err := prompt.MultiSelect("Choose options:", options)

		assert.Error(t, err)  // Invalid choice should return error
		assert.Nil(t, result) // Should return nil on error
	})
}

// TestPromptPassword tests the Password functionality
func TestPromptPassword(t *testing.T) {
	t.Run("Password_basic", func(t *testing.T) {
		input := "secretpassword\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Password("Enter password:")
		assert.NoError(t, err)
		assert.Equal(t, "secretpassword", result)
	})

	t.Run("Password_empty", func(t *testing.T) {
		input := "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Password("Enter password:")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

// TestPromptErrorHandling tests error handling scenarios
func TestPromptErrorHandling(t *testing.T) {
	t.Run("reader_error", func(t *testing.T) {
		errorReader := &ErrorReader{}
		prompt := &Prompt{
			reader:    bufio.NewReader(errorReader),
			formatter: NewOutputFormatter(),
		}

		// Should handle reader errors gracefully
		result := prompt.Confirm("Test?", true)
		assert.True(t, result) // Should return default
	})

	t.Run("empty_input_stream", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("")),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "")
		assert.Error(t, err) // EOF should return error
		assert.Empty(t, result)
	})
}

// TestPromptEdgeCases tests edge cases
func TestPromptEdgeCases(t *testing.T) {
	t.Run("very_long_input", func(t *testing.T) {
		longInput := strings.Repeat("a", 1000) + "\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(longInput)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "")
		assert.NoError(t, err)
		assert.Equal(t, strings.Repeat("a", 1000), result)
	})

	t.Run("unicode_input", func(t *testing.T) {
		unicodeInput := "测试输入 🚀\n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(unicodeInput)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "")
		assert.NoError(t, err)
		assert.Equal(t, "测试输入 🚀", result)
	})

	t.Run("whitespace_only_input", func(t *testing.T) {
		input := "   \n"
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader(input)),
			formatter: NewOutputFormatter(),
		}

		result, err := prompt.Input("Enter something:", "")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

// Helper types for testing

type ErrorReader struct{}

func (r *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
