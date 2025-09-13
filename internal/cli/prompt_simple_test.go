package cli

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrompt_BasicMethods(t *testing.T) {
	t.Run("NewPrompt", func(t *testing.T) {
		prompt := NewPrompt()
		assert.NotNil(t, prompt)
		assert.NotNil(t, prompt.reader)
		assert.NotNil(t, prompt.formatter)
	})

	t.Run("Confirm with yes", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("y\n")),
			formatter: NewOutputFormatter(),
		}
		result := prompt.Confirm("Continue?", false)
		assert.True(t, result)
	})

	t.Run("Confirm with no", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("n\n")),
			formatter: NewOutputFormatter(),
		}
		result := prompt.Confirm("Continue?", false)
		assert.False(t, result)
	})

	t.Run("Confirm with default", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("\n")),
			formatter: NewOutputFormatter(),
		}
		result := prompt.Confirm("Continue?", true)
		assert.True(t, result)
	})

	t.Run("Select option", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("2\n")),
			formatter: NewOutputFormatter(),
		}
		index, err := prompt.Select("Choose", []string{"Option 1", "Option 2", "Option 3"})
		assert.NoError(t, err)
		assert.Equal(t, 1, index)
	})

	t.Run("MultiSelect options", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("1,3\n")),
			formatter: NewOutputFormatter(),
		}
		indices, err := prompt.MultiSelect("Choose", []string{"Option 1", "Option 2", "Option 3"})
		assert.NoError(t, err)
		assert.Equal(t, []int{0, 2}, indices)
	})

	t.Run("Input with value", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("test value\n")),
			formatter: NewOutputFormatter(),
		}
		result, err := prompt.Input("Enter value", "")
		assert.NoError(t, err)
		assert.Equal(t, "test value", result)
	})

	t.Run("Input with default", func(t *testing.T) {
		prompt := &Prompt{
			reader:    bufio.NewReader(strings.NewReader("\n")),
			formatter: NewOutputFormatter(),
		}
		result, err := prompt.Input("Enter value", "default")
		assert.NoError(t, err)
		assert.Equal(t, "default", result)
	})
}