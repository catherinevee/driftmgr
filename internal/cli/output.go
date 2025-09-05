package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// OutputFormat defines the output format type
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatYAML  OutputFormat = "yaml"
	FormatTree  OutputFormat = "tree"
	FormatPlain OutputFormat = "plain"
)

// Color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
	
	ColorBold      = "\033[1m"
	ColorUnderline = "\033[4m"
)

// OutputFormatter handles formatted output
type OutputFormatter struct {
	writer      io.Writer
	format      OutputFormat
	noColor     bool
	showHeaders bool
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{
		writer:      os.Stdout,
		format:      FormatTable,
		noColor:     false,
		showHeaders: true,
	}
}

// SetFormat sets the output format
func (f *OutputFormatter) SetFormat(format OutputFormat) {
	f.format = format
}

// DisableColor disables colored output
func (f *OutputFormatter) DisableColor() {
	f.noColor = true
}

// Color returns colored string if color is enabled
func (f *OutputFormatter) Color(text, color string) string {
	if f.noColor {
		return text
	}
	return color + text + ColorReset
}

// Success prints a success message
func (f *OutputFormatter) Success(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintln(f.writer, f.Color("✓ "+msg, ColorGreen))
}

// Error prints an error message
func (f *OutputFormatter) Error(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintln(f.writer, f.Color("✗ "+msg, ColorRed))
}

// Warning prints a warning message
func (f *OutputFormatter) Warning(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintln(f.writer, f.Color("⚠ "+msg, ColorYellow))
}

// Info prints an info message
func (f *OutputFormatter) Info(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintln(f.writer, f.Color("ℹ "+msg, ColorBlue))
}

// Header prints a header
func (f *OutputFormatter) Header(text string) {
	fmt.Fprintln(f.writer)
	fmt.Fprintln(f.writer, f.Color(strings.ToUpper(text), ColorBold+ColorCyan))
	fmt.Fprintln(f.writer, f.Color(strings.Repeat("=", len(text)), ColorCyan))
}

// Section prints a section header
func (f *OutputFormatter) Section(text string) {
	fmt.Fprintln(f.writer)
	fmt.Fprintln(f.writer, f.Color(text, ColorBold))
	fmt.Fprintln(f.writer, f.Color(strings.Repeat("-", len(text)), ColorGray))
}

// Table prints data in table format
func (f *OutputFormatter) Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	
	// Print headers
	if f.showHeaders && len(headers) > 0 {
		headerLine := strings.Join(headers, "\t")
		fmt.Fprintln(w, f.Color(headerLine, ColorBold))
		
		// Print separator
		separators := make([]string, len(headers))
		for i, header := range headers {
			separators[i] = strings.Repeat("-", len(header))
		}
		fmt.Fprintln(w, strings.Join(separators, "\t"))
	}
	
	// Print rows
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	
	w.Flush()
}

// Tree prints data in tree format
func (f *OutputFormatter) Tree(root TreeNode) {
	f.printTreeNode(root, "", true)
}

// printTreeNode recursively prints tree nodes
func (f *OutputFormatter) printTreeNode(node TreeNode, prefix string, isLast bool) {
	// Determine the connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	
	// Print current node
	if prefix == "" {
		fmt.Fprintln(f.writer, node.GetName())
	} else {
		fmt.Fprintf(f.writer, "%s%s%s\n", prefix, connector, node.GetName())
	}
	
	// Determine the prefix for children
	childPrefix := prefix
	if prefix == "" {
		// Root node
		childPrefix = ""
	} else if isLast {
		childPrefix = prefix + "    "
	} else {
		childPrefix = prefix + "│   "
	}
	
	// Print children
	children := node.GetChildren()
	for i, child := range children {
		f.printTreeNode(child, childPrefix, i == len(children)-1)
	}
}

// TreeNode interface for tree printing
type TreeNode interface {
	GetName() string
	GetChildren() []TreeNode
}

// SimpleTreeNode is a basic implementation of TreeNode
type SimpleTreeNode struct {
	Name     string
	Children []TreeNode
}

func (n SimpleTreeNode) GetName() string {
	return n.Name
}

func (n SimpleTreeNode) GetChildren() []TreeNode {
	return n.Children
}

// KeyValue prints a key-value pair
func (f *OutputFormatter) KeyValue(key, value string) {
	fmt.Fprintf(f.writer, "%s: %s\n", 
		f.Color(key, ColorBold),
		value)
}

// KeyValueList prints a list of key-value pairs
func (f *OutputFormatter) KeyValueList(items map[string]string) {
	maxKeyLen := 0
	for key := range items {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}
	
	for key, value := range items {
		padding := strings.Repeat(" ", maxKeyLen-len(key))
		fmt.Fprintf(f.writer, "%s%s : %s\n",
			f.Color(key, ColorBold),
			padding,
			value)
	}
}

// ProgressBar prints a simple progress bar
func (f *OutputFormatter) ProgressBar(current, total int, width int) {
	if total <= 0 {
		return
	}
	
	percent := float64(current) / float64(total)
	filled := int(float64(width) * percent)
	
	bar := strings.Builder{}
	bar.WriteString("[")
	
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString("=")
		} else if i == filled {
			bar.WriteString(">")
		} else {
			bar.WriteString(" ")
		}
	}
	
	bar.WriteString("] ")
	bar.WriteString(fmt.Sprintf("%.1f%%", percent*100))
	
	fmt.Fprint(f.writer, "\r"+bar.String())
	if current >= total {
		fmt.Fprintln(f.writer)
	}
}

// List prints a list with bullets
func (f *OutputFormatter) List(items []string) {
	for _, item := range items {
		fmt.Fprintf(f.writer, "  • %s\n", item)
	}
}

// NumberedList prints a numbered list
func (f *OutputFormatter) NumberedList(items []string) {
	for i, item := range items {
		fmt.Fprintf(f.writer, "  %d. %s\n", i+1, item)
	}
}

// Box prints text in a box
func (f *OutputFormatter) Box(title, content string) {
	lines := strings.Split(content, "\n")
	maxLen := len(title)
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}
	
	// Top border
	fmt.Fprintf(f.writer, "┌─%s─┐\n", strings.Repeat("─", maxLen))
	
	// Title
	if title != "" {
		padding := strings.Repeat(" ", maxLen-len(title))
		fmt.Fprintf(f.writer, "│ %s%s │\n", f.Color(title, ColorBold), padding)
		fmt.Fprintf(f.writer, "├─%s─┤\n", strings.Repeat("─", maxLen))
	}
	
	// Content
	for _, line := range lines {
		padding := strings.Repeat(" ", maxLen-len(line))
		fmt.Fprintf(f.writer, "│ %s%s │\n", line, padding)
	}
	
	// Bottom border
	fmt.Fprintf(f.writer, "└─%s─┘\n", strings.Repeat("─", maxLen))
}

// StatusIcon returns an appropriate status icon
func (f *OutputFormatter) StatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "success", "complete", "ok":
		return f.Color("✓", ColorGreen)
	case "error", "failed", "fail":
		return f.Color("✗", ColorRed)
	case "warning", "warn":
		return f.Color("⚠", ColorYellow)
	case "info":
		return f.Color("ℹ", ColorBlue)
	case "pending", "running":
		return f.Color("◎", ColorCyan)
	case "skipped":
		return f.Color("⊘", ColorGray)
	default:
		return "•"
	}
}