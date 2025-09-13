package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Prompt provides interactive prompting capabilities
type Prompt struct {
	reader    *bufio.Reader
	formatter *OutputFormatter
}

// NewPrompt creates a new prompt
func NewPrompt() *Prompt {
	return &Prompt{
		reader:    bufio.NewReader(os.Stdin),
		formatter: NewOutputFormatter(),
	}
}

// Confirm asks for yes/no confirmation
func (p *Prompt) Confirm(message string, defaultYes bool) bool {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", message, defaultStr)

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// ConfirmWithDetails asks for confirmation with additional details
func (p *Prompt) ConfirmWithDetails(message string, details []string) bool {
	p.formatter.Warning("%s", message)

	if len(details) > 0 {
		fmt.Println("\nDetails:")
		for _, detail := range details {
			fmt.Printf("  • %s\n", detail)
		}
		fmt.Println()
	}

	return p.Confirm("Do you want to proceed?", false)
}

// Select asks the user to select from a list of options
func (p *Prompt) Select(message string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	fmt.Println(message)
	fmt.Println()

	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}

	fmt.Printf("\nSelect option [1-%d]: ", len(options))

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return -1, err
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid input: %s", input)
	}

	if choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("invalid selection: %d", choice)
	}

	return choice - 1, nil
}

// MultiSelect asks the user to select multiple options
func (p *Prompt) MultiSelect(message string, options []string) ([]int, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options provided")
	}

	fmt.Println(message)
	fmt.Println()

	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}

	fmt.Println("\nEnter options separated by commas (e.g., 1,3,5)")
	fmt.Printf("Select options [1-%d]: ", len(options))

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return []int{}, nil
	}

	parts := strings.Split(input, ",")
	selections := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		choice, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid input: %s", part)
		}

		if choice < 1 || choice > len(options) {
			return nil, fmt.Errorf("invalid selection: %d", choice)
		}

		selections = append(selections, choice-1)
	}

	return selections, nil
}

// Input asks for text input
func (p *Prompt) Input(message string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return input, nil
}

// Password asks for password input (Note: input will be visible)
func (p *Prompt) Password(message string) (string, error) {
	fmt.Printf("%s: ", message)

	// Note: For production, use a proper password input library
	// that masks the input. This is a simple implementation.
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}

// ConfirmDangerous asks for confirmation of dangerous operations
func (p *Prompt) ConfirmDangerous(operation string, resources []string) bool {
	p.formatter.Error("DANGEROUS OPERATION: %s", operation)
	fmt.Println()

	if len(resources) > 0 {
		p.formatter.Warning("This will affect the following resources:")
		for _, resource := range resources {
			fmt.Printf("  • %s\n", resource)
		}
		fmt.Println()
	}

	// First confirmation
	if !p.Confirm("Are you sure you want to proceed?", false) {
		return false
	}

	// Second confirmation for extra safety
	fmt.Println()
	confirmation, err := p.Input("Type 'yes' to confirm", "")
	if err != nil {
		return false
	}

	return strings.ToLower(confirmation) == "yes"
}

// SelectWithFilter allows filtering of options
func (p *Prompt) SelectWithFilter(message string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	fmt.Println(message)
	fmt.Println()

	// First, ask if they want to filter
	filter, _ := p.Input("Enter filter (or press Enter to show all)", "")

	filteredOptions := make([]string, 0)
	filteredIndices := make([]int, 0)

	if filter == "" {
		filteredOptions = options
		for i := range options {
			filteredIndices = append(filteredIndices, i)
		}
	} else {
		filter = strings.ToLower(filter)
		for i, option := range options {
			if strings.Contains(strings.ToLower(option), filter) {
				filteredOptions = append(filteredOptions, option)
				filteredIndices = append(filteredIndices, i)
			}
		}
	}

	if len(filteredOptions) == 0 {
		return -1, fmt.Errorf("no options match filter: %s", filter)
	}

	// Display filtered options
	for i, option := range filteredOptions {
		fmt.Printf("  %d) %s\n", i+1, option)
	}

	fmt.Printf("\nSelect option [1-%d]: ", len(filteredOptions))

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return -1, err
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid input: %s", input)
	}

	if choice < 1 || choice > len(filteredOptions) {
		return -1, fmt.Errorf("invalid selection: %d", choice)
	}

	return filteredIndices[choice-1], nil
}

// TableSelect shows options in a table and allows selection
func (p *Prompt) TableSelect(message string, headers []string, rows [][]string) (int, error) {
	if len(rows) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	fmt.Println(message)
	fmt.Println()

	// Add row numbers
	numberedHeaders := append([]string{"#"}, headers...)
	numberedRows := make([][]string, len(rows))
	for i, row := range rows {
		numberedRows[i] = append([]string{fmt.Sprintf("%d", i+1)}, row...)
	}

	p.formatter.Table(numberedHeaders, numberedRows)

	fmt.Printf("\nSelect option [1-%d]: ", len(rows))

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return -1, err
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid input: %s", input)
	}

	if choice < 1 || choice > len(rows) {
		return -1, fmt.Errorf("invalid selection: %d", choice)
	}

	return choice - 1, nil
}
