package utils

import (
	"fmt"
	"strings"
)

// GetString safely extracts a string value from a map
func GetString(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// SplitPath splits a path string into parts with a maximum number of parts
func SplitPath(path, separator string, maxParts int) []string {
	parts := strings.SplitN(path, separator, maxParts)

	// Filter out empty parts
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// SafeString returns a string value or empty string if nil
func SafeString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

// Contains checks if a slice contains a specific value
func Contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Unique removes duplicates from a string slice
func Unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
