package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/catherinevee/driftmgr/internal/models"
)

// CompletionData holds all data needed for auto-completion
type CompletionData struct {
	Commands              []string
	Providers             []string
	Regions               map[string][]string
	StateFiles            []string
	Resources             []string
	ResourceTypes         []string
	ResourceNames         []string
	FilePaths             []string
	NotificationTypes     []string
	ExportFormats         []string
	SeverityLevels        []string
	RemediationStrategies []string
}

// NewCompletionData creates a new completion data structure with default values
func NewCompletionData() *CompletionData {
	return &CompletionData{
		Commands: []string{
			"discover", "analyze", "perspective", "visualize", "diagram", "export", "statefiles",
			"credentials", "remediate", "remediate-batch", "remediate-history",
			"remediate-rollback", "health", "notify", "terragrunt",
			"help", "history", "clear", "exit", "quit",
		},
		Providers: []string{"aws", "azure", "gcp"},
		Regions: map[string][]string{
			"aws": {
				"us-east-1", "us-east-2", "us-west-1", "us-west-2",
				"eu-west-1", "eu-central-1", "ap-southeast-1", "ap-northeast-1",
				"sa-east-1", "ca-central-1", "af-south-1", "me-south-1",
			},
			"azure": {
				"eastus", "eastus2", "southcentralus", "westus2",
				"westus3", "canadacentral", "northeurope", "westeurope",
				"uksouth", "ukwest", "japaneast", "japanwest",
			},
			"gcp": {
				"us-central1", "us-east1", "us-west1", "europe-west1",
				"asia-east1", "asia-northeast1", "australia-southeast1",
			},
		},
		NotificationTypes:     []string{"email", "slack", "webhook"},
		ExportFormats:         []string{"png", "svg", "pdf", "json", "yaml"},
		SeverityLevels:        []string{"low", "medium", "high", "critical"},
		RemediationStrategies: []string{"auto", "manual", "dry-run", "generate"},
	}
}

// EnhancedInputReader provides advanced input reading with completion
type EnhancedInputReader struct {
	reader         *bufio.Reader
	completionData *CompletionData
	history        []string
	historyIndex   int
	lineBuffer     string
	cursorPos      int
}

// NewEnhancedInputReader creates a new enhanced input reader
func NewEnhancedInputReader() *EnhancedInputReader {
	return &EnhancedInputReader{
		reader:         bufio.NewReader(os.Stdin),
		completionData: NewCompletionData(),
		history:        make([]string, 0),
		historyIndex:   -1,
		lineBuffer:     "",
		cursorPos:      0,
	}
}

// ReadLine reads a line with enhanced features
func (eir *EnhancedInputReader) ReadLine() (string, error) {
	fmt.Printf("%sdriftmgr> %s", ColorGreen, ColorReset)

	eir.lineBuffer = ""
	eir.cursorPos = 0

	for {
		char, _, err := eir.reader.ReadRune()
		if err != nil {
			return "", err
		}

		switch char {
		case '\n', '\r':
			fmt.Println()
			return eir.lineBuffer, nil
		case '\t':
			eir.handleTabCompletion()
		case 127: // Backspace
			eir.handleBackspace()
		case 27: // Escape sequence
			eir.handleEscapeSequence()
		default:
			if unicode.IsPrint(char) {
				eir.insertChar(char)
			}
		}
	}
}

// handleTabCompletion handles tab completion
func (eir *EnhancedInputReader) handleTabCompletion() {
	words := strings.Fields(eir.lineBuffer)
	if len(words) == 0 {
		return
	}

	currentWord := eir.getCurrentWord()
	wordIndex := eir.getCurrentWordIndex()

	var completions []string

	if wordIndex == 0 {
		// Command completion
		completions = eir.fuzzySearch(eir.completionData.Commands, currentWord)
	} else if wordIndex == 1 {
		// First argument completion
		command := words[0]
		completions = eir.getCompletionsForCommand(command, currentWord)
	} else {
		// Subsequent argument completion
		command := words[0]
		completions = eir.getCompletionsForArgument(command, wordIndex, currentWord)
	}

	if len(completions) == 1 {
		// Single completion - complete it
		eir.completeWord(completions[0])
	} else if len(completions) > 1 {
		// Multiple completions - show options
		eir.showCompletions(completions)
	}
}

// getCompletionsForCommand returns completions for the first argument of a command
func (eir *EnhancedInputReader) getCompletionsForCommand(command, currentWord string) []string {
	switch command {
	case "discover":
		return eir.fuzzySearch(eir.completionData.Providers, currentWord)
	case "analyze", "perspective", "visualize", "diagram":
		return eir.fuzzySearch(eir.completionData.StateFiles, currentWord)
	case "export":
		return eir.fuzzySearch(eir.completionData.ExportFormats, currentWord)
	case "remediate":
		return eir.fuzzySearch(eir.completionData.Resources, currentWord)
	case "notify":
		return eir.fuzzySearch(eir.completionData.NotificationTypes, currentWord)
	case "credentials":
		return eir.fuzzySearch([]string{"setup", "list", "validate"}, currentWord)
	case "terragrunt":
		return eir.fuzzySearch([]string{"files", "statefiles", "analyze"}, currentWord)
	default:
		return []string{}
	}
}

// getCompletionsForArgument returns completions for subsequent arguments
func (eir *EnhancedInputReader) getCompletionsForArgument(command string, argIndex int, currentWord string) []string {
	switch command {
	case "discover":
		if argIndex == 1 {
			// Provider was specified, now suggest regions
			provider := strings.Fields(eir.lineBuffer)[1]
			if regions, exists := eir.completionData.Regions[provider]; exists {
				return eir.fuzzySearch(regions, currentWord)
			}
		}
	case "perspective":
		if argIndex == 2 {
			return eir.fuzzySearch(eir.completionData.Providers, currentWord)
		}
	case "export":
		if argIndex == 2 {
			return eir.fuzzySearch(eir.completionData.ExportFormats, currentWord)
		}
	case "notify":
		if argIndex == 2 {
			// Subject suggestions
			return eir.fuzzySearch([]string{"Drift Alert", "Security Alert", "Resource Change", "Compliance Issue"}, currentWord)
		}
	case "remediate":
		if strings.HasPrefix(currentWord, "--") {
			return eir.fuzzySearch(eir.completionData.RemediationStrategies, currentWord)
		}
	}

	return []string{}
}

// fuzzySearch performs fuzzy search on a list of strings
func (eir *EnhancedInputReader) fuzzySearch(items []string, query string) []string {
	if query == "" {
		return items
	}

	var results []string
	queryLower := strings.ToLower(query)

	for _, item := range items {
		itemLower := strings.ToLower(item)
		if strings.Contains(itemLower, queryLower) || strings.HasPrefix(itemLower, queryLower) {
			results = append(results, item)
		}
	}

	return results
}

// getCurrentWord returns the current word being typed
func (eir *EnhancedInputReader) getCurrentWord() string {
	words := strings.Fields(eir.lineBuffer)
	if len(words) == 0 {
		return ""
	}

	// Find the current word based on cursor position
	currentPos := 0
	for _, word := range words {
		if currentPos <= eir.cursorPos && eir.cursorPos <= currentPos+len(word) {
			return word
		}
		currentPos += len(word) + 1 // +1 for space
	}

	return ""
}

// getCurrentWordIndex returns the index of the current word
func (eir *EnhancedInputReader) getCurrentWordIndex() int {
	words := strings.Fields(eir.lineBuffer)
	if len(words) == 0 {
		return 0
	}

	currentPos := 0
	for i, word := range words {
		if currentPos <= eir.cursorPos && eir.cursorPos <= currentPos+len(word) {
			return i
		}
		currentPos += len(word) + 1 // +1 for space
	}

	return len(words)
}

// completeWord completes the current word with the given completion
func (eir *EnhancedInputReader) completeWord(completion string) {
	words := strings.Fields(eir.lineBuffer)
	if len(words) == 0 {
		eir.lineBuffer = completion
		eir.cursorPos = len(completion)
		eir.redrawLine()
		return
	}

	// Find and replace the current word
	currentPos := 0
	for i, word := range words {
		if currentPos <= eir.cursorPos && eir.cursorPos <= currentPos+len(word) {
			words[i] = completion
			break
		}
		currentPos += len(word) + 1
	}

	eir.lineBuffer = strings.Join(words, " ")
	eir.cursorPos = len(eir.lineBuffer)
	eir.redrawLine()
}

// showCompletions displays available completions
func (eir *EnhancedInputReader) showCompletions(completions []string) {
	fmt.Println()
	fmt.Printf("%sAvailable completions:%s\n", ColorCyan, ColorReset)

	// Sort completions for consistent display
	sort.Strings(completions)

	// Display in columns
	cols := 3
	for i := 0; i < len(completions); i += cols {
		end := i + cols
		if end > len(completions) {
			end = len(completions)
		}

		row := completions[i:end]
		for j, completion := range row {
			fmt.Printf("%s%-20s%s", ColorGreen, completion, ColorReset)
			if j < len(row)-1 {
				fmt.Print("  ")
			}
		}
		fmt.Println()
	}

	fmt.Printf("%sdriftmgr> %s%s", ColorGreen, ColorReset, eir.lineBuffer)
}

// handleBackspace handles backspace key
func (eir *EnhancedInputReader) handleBackspace() {
	if eir.cursorPos > 0 {
		eir.lineBuffer = eir.lineBuffer[:eir.cursorPos-1] + eir.lineBuffer[eir.cursorPos:]
		eir.cursorPos--
		eir.redrawLine()
	}
}

// handleEscapeSequence handles escape sequences (arrow keys, etc.)
func (eir *EnhancedInputReader) handleEscapeSequence() {
	// Read the next character to determine the sequence
	char, _, err := eir.reader.ReadRune()
	if err != nil {
		return
	}

	if char == '[' {
		// Arrow key sequence
		char, _, err = eir.reader.ReadRune()
		if err != nil {
			return
		}

		switch char {
		case 'A': // Up arrow - history
			eir.navigateHistory(-1)
		case 'B': // Down arrow - history
			eir.navigateHistory(1)
		case 'C': // Right arrow
			if eir.cursorPos < len(eir.lineBuffer) {
				eir.cursorPos++
				eir.redrawLine()
			}
		case 'D': // Left arrow
			if eir.cursorPos > 0 {
				eir.cursorPos--
				eir.redrawLine()
			}
		}
	}
}

// navigateHistory navigates through command history
func (eir *EnhancedInputReader) navigateHistory(direction int) {
	if len(eir.history) == 0 {
		return
	}

	eir.historyIndex += direction
	if eir.historyIndex >= len(eir.history) {
		eir.historyIndex = len(eir.history) - 1
	} else if eir.historyIndex < 0 {
		eir.historyIndex = 0
	}

	eir.lineBuffer = eir.history[eir.historyIndex]
	eir.cursorPos = len(eir.lineBuffer)
	eir.redrawLine()
}

// insertChar inserts a character at the current cursor position
func (eir *EnhancedInputReader) insertChar(char rune) {
	if eir.cursorPos == len(eir.lineBuffer) {
		eir.lineBuffer += string(char)
	} else {
		eir.lineBuffer = eir.lineBuffer[:eir.cursorPos] + string(char) + eir.lineBuffer[eir.cursorPos:]
	}
	eir.cursorPos++
	eir.redrawLine()
}

// redrawLine redraws the current line
func (eir *EnhancedInputReader) redrawLine() {
	// Clear the current line
	fmt.Print("\r\033[K")
	fmt.Printf("%sdriftmgr> %s%s", ColorGreen, ColorReset, eir.lineBuffer)

	// Position cursor correctly
	if eir.cursorPos < len(eir.lineBuffer) {
		fmt.Printf("\033[%dD", len(eir.lineBuffer)-eir.cursorPos)
	}
}

// UpdateCompletionData updates completion data with discovered resources
func (eir *EnhancedInputReader) UpdateCompletionData(resources []models.Resource) {
	var resourceNames, resourceTypes []string
	seenTypes := make(map[string]bool)
	seenNames := make(map[string]bool)

	for _, resource := range resources {
		if !seenNames[resource.Name] {
			resourceNames = append(resourceNames, resource.Name)
			seenNames[resource.Name] = true
		}
		if !seenTypes[resource.Type] {
			resourceTypes = append(resourceTypes, resource.Type)
			seenTypes[resource.Type] = true
		}
	}

	eir.completionData.ResourceNames = resourceNames
	eir.completionData.ResourceTypes = resourceTypes
}

// UpdateStateFiles updates completion data with available state files
func (eir *EnhancedInputReader) UpdateStateFiles(stateFiles []models.StateFile) {
	var fileNames []string
	for _, sf := range stateFiles {
		fileNames = append(fileNames, sf.Path)
	}
	eir.completionData.StateFiles = fileNames
}

// AddToHistory adds a command to history
func (eir *EnhancedInputReader) AddToHistory(command string) {
	if command == "" {
		return
	}

	// Don't add duplicate consecutive commands
	if len(eir.history) > 0 && eir.history[len(eir.history)-1] == command {
		return
	}

	eir.history = append(eir.history, command)
	if len(eir.history) > 100 {
		eir.history = eir.history[1:]
	}
	eir.historyIndex = -1
}

// GetSuggestions returns auto-suggestions based on history
func (eir *EnhancedInputReader) GetSuggestions(partial string) []string {
	if partial == "" {
		return []string{}
	}

	var suggestions []string
	partialLower := strings.ToLower(partial)

	// Search in history
	for _, cmd := range eir.history {
		if strings.HasPrefix(strings.ToLower(cmd), partialLower) {
			suggestions = append(suggestions, cmd)
		}
	}

	// Search in commands
	for _, cmd := range eir.completionData.Commands {
		if strings.HasPrefix(strings.ToLower(cmd), partialLower) {
			suggestions = append(suggestions, cmd)
		}
	}

	// Remove duplicates and limit results
	seen := make(map[string]bool)
	var uniqueSuggestions []string
	for _, suggestion := range suggestions {
		if !seen[suggestion] {
			uniqueSuggestions = append(uniqueSuggestions, suggestion)
			seen[suggestion] = true
		}
	}

	if len(uniqueSuggestions) > 5 {
		uniqueSuggestions = uniqueSuggestions[:5]
	}

	return uniqueSuggestions
}
