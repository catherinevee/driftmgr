package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputFormatter(t *testing.T) {
	t.Run("ColorOutput", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf

		// Test with color enabled (default)
		colored := formatter.Color("test", ColorGreen)
		assert.Contains(t, colored, ColorGreen)
		assert.Contains(t, colored, "test")
		assert.Contains(t, colored, ColorReset)

		// Test with color disabled
		formatter.DisableColor()
		plain := formatter.Color("test", ColorGreen)
		assert.Equal(t, "test", plain)
	})

	t.Run("MessageTypes", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf
		formatter.DisableColor() // Disable color for easier testing

		formatter.Success("Operation completed")
		assert.Contains(t, buf.String(), "✓ Operation completed")

		buf.Reset()
		formatter.Error("Operation failed")
		assert.Contains(t, buf.String(), "✗ Operation failed")

		buf.Reset()
		formatter.Warning("Check configuration")
		assert.Contains(t, buf.String(), "⚠ Check configuration")

		buf.Reset()
		formatter.Info("Processing data")
		assert.Contains(t, buf.String(), "ℹ Processing data")
	})

	t.Run("Headers", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf
		formatter.DisableColor()

		formatter.Header("Main Title")
		output := buf.String()
		assert.Contains(t, output, "MAIN TITLE")
		assert.Contains(t, output, "==========")

		buf.Reset()
		formatter.Section("Subsection")
		output = buf.String()
		assert.Contains(t, output, "Subsection")
		assert.Contains(t, output, "----------")
	})

	t.Run("Table", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf
		formatter.DisableColor()

		headers := []string{"Name", "Type", "Status"}
		rows := [][]string{
			{"resource1", "aws_instance", "active"},
			{"resource2", "aws_s3_bucket", "pending"},
		}

		formatter.Table(headers, rows)
		output := buf.String()

		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "Type")
		assert.Contains(t, output, "Status")
		assert.Contains(t, output, "resource1")
		assert.Contains(t, output, "aws_instance")
		assert.Contains(t, output, "active")
	})

	t.Run("KeyValueList", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf
		formatter.DisableColor()

		items := map[string]string{
			"Provider":  "AWS",
			"Region":    "us-east-1",
			"Resources": "42",
		}

		formatter.KeyValueList(items)
		output := buf.String()

		assert.Contains(t, output, "Provider")
		assert.Contains(t, output, "AWS")
		assert.Contains(t, output, "Region")
		assert.Contains(t, output, "us-east-1")
		assert.Contains(t, output, "Resources")
		assert.Contains(t, output, "42")
	})

	t.Run("Tree", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf

		// Create a simple tree structure
		root := SimpleTreeNode{
			Name: "root",
			Children: []TreeNode{
				SimpleTreeNode{
					Name: "child1",
					Children: []TreeNode{
						SimpleTreeNode{Name: "grandchild1"},
						SimpleTreeNode{Name: "grandchild2"},
					},
				},
				SimpleTreeNode{
					Name: "child2",
				},
			},
		}

		formatter.Tree(root)
		output := buf.String()

		// Tree should at minimum contain all node names
		assert.Contains(t, output, "root")
		assert.Contains(t, output, "child1")
		assert.Contains(t, output, "grandchild1")
		assert.Contains(t, output, "grandchild2")
		assert.Contains(t, output, "child2")
	})

	t.Run("StatusIcons", func(t *testing.T) {
		formatter := NewOutputFormatter()
		formatter.DisableColor()

		assert.Equal(t, "✓", formatter.StatusIcon("success"))
		assert.Equal(t, "✗", formatter.StatusIcon("error"))
		assert.Equal(t, "⚠", formatter.StatusIcon("warning"))
		assert.Equal(t, "ℹ", formatter.StatusIcon("info"))
		assert.Equal(t, "◎", formatter.StatusIcon("running"))
		assert.Equal(t, "⊘", formatter.StatusIcon("skipped"))
		assert.Equal(t, "•", formatter.StatusIcon("unknown"))
	})

	t.Run("Lists", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf

		items := []string{"First item", "Second item", "Third item"}

		formatter.List(items)
		output := buf.String()
		assert.Contains(t, output, "• First item")
		assert.Contains(t, output, "• Second item")
		assert.Contains(t, output, "• Third item")

		buf.Reset()
		formatter.NumberedList(items)
		output = buf.String()
		assert.Contains(t, output, "1. First item")
		assert.Contains(t, output, "2. Second item")
		assert.Contains(t, output, "3. Third item")
	})

	t.Run("Box", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf
		formatter.DisableColor()

		formatter.Box("Title", "Content line 1\nContent line 2")
		output := buf.String()

		assert.Contains(t, output, "┌─")
		assert.Contains(t, output, "─┐")
		assert.Contains(t, output, "│ Title")
		assert.Contains(t, output, "├─")
		assert.Contains(t, output, "─┤")
		assert.Contains(t, output, "│ Content line 1")
		assert.Contains(t, output, "│ Content line 2")
		assert.Contains(t, output, "└─")
		assert.Contains(t, output, "─┘")
	})

	t.Run("ProgressBar", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter()
		formatter.writer = &buf

		formatter.ProgressBar(50, 100, 20)
		output := buf.String()

		assert.Contains(t, output, "[")
		assert.Contains(t, output, "]")
		assert.Contains(t, output, "50.0%")
		assert.Contains(t, output, "=")
		assert.Contains(t, output, ">")
	})
}

func TestFormatValidation(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test format setting
	formatter.SetFormat(FormatJSON)
	assert.Equal(t, FormatJSON, formatter.format)

	formatter.SetFormat(FormatTable)
	assert.Equal(t, FormatTable, formatter.format)
}

func TestFormattingWithSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewOutputFormatter()
	formatter.writer = &buf
	formatter.DisableColor()

	// Test with special characters
	formatter.Success("Success with 特殊文字 and émojis 🎉")
	assert.Contains(t, buf.String(), "特殊文字")
	assert.Contains(t, buf.String(), "émojis")
	assert.Contains(t, buf.String(), "🎉")

	// Test with long strings
	buf.Reset()
	longString := strings.Repeat("a", 200)
	formatter.Info(longString)
	assert.Contains(t, buf.String(), longString)
}
