package api

import (
	"context"
	// "encoding/json" // Not used
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIVersion represents a supported API version
type APIVersion struct {
	Major           int        `json:"major"`
	Minor           int        `json:"minor"`
	Patch           int        `json:"patch"`
	Status          string     `json:"status"` // stable, beta, alpha, deprecated
	ReleaseDate     time.Time  `json:"release_date"`
	DeprecationDate *time.Time `json:"deprecation_date,omitempty"`
	SunsetDate      *time.Time `json:"sunset_date,omitempty"`
	Description     string     `json:"description"`
	ChangeLog       []string   `json:"changelog,omitempty"`
}

// String returns the version as a string (e.g., "1.2.3")
func (v APIVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsCompatible checks if this version is compatible with the requested version
func (v APIVersion) IsCompatible(requested APIVersion) bool {
	// Major version must match for compatibility
	if v.Major != requested.Major {
		return false
	}

	// This version must be >= requested version for minor/patch
	if v.Minor < requested.Minor {
		return false
	}

	if v.Minor == requested.Minor && v.Patch < requested.Patch {
		return false
	}

	return true
}

// IsDeprecated checks if the version is deprecated
func (v APIVersion) IsDeprecated() bool {
	return v.Status == "deprecated" ||
		(v.DeprecationDate != nil && time.Now().After(*v.DeprecationDate))
}

// IsSunset checks if the version is past its sunset date
func (v APIVersion) IsSunset() bool {
	return v.SunsetDate != nil && time.Now().After(*v.SunsetDate)
}

// VersionManager manages API versions and compatibility
type VersionManager struct {
	supportedVersions map[string]APIVersion
	defaultVersion    string
	latestVersion     string
	compatibilityMap  map[string][]string // version -> compatible versions
}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	vm := &VersionManager{
		supportedVersions: make(map[string]APIVersion),
		compatibilityMap:  make(map[string][]string),
	}

	// Initialize with current supported versions
	vm.initializeVersions()

	return vm
}

// initializeVersions sets up the supported API versions
func (vm *VersionManager) initializeVersions() {
	// Version 1.0.0 - Initial stable release
	v1_0_0 := APIVersion{
		Major: 1, Minor: 0, Patch: 0,
		Status:      "stable",
		ReleaseDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Description: "Initial stable API release",
		ChangeLog: []string{
			"Basic resource discovery",
			"Multi-cloud provider support",
			"Account management",
		},
	}

	// Version 1.1.0 - Plugin support
	v1_1_0 := APIVersion{
		Major: 1, Minor: 1, Patch: 0,
		Status:      "stable",
		ReleaseDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Description: "Added plugin architecture support",
		ChangeLog: []string{
			"Plugin architecture",
			"Dynamic provider loading",
			"Enhanced configuration system",
		},
	}

	// Version 2.0.0 - Major rewrite with microservices
	v2_0_0 := APIVersion{
		Major: 2, Minor: 0, Patch: 0,
		Status:      "beta",
		ReleaseDate: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
		Description: "Microservices architecture and streaming APIs",
		ChangeLog: []string{
			"Microservices architecture",
			"Streaming APIs",
			"Advanced caching",
			"GraphQL support",
		},
	}

	// Version 3.0.0 - Future version with AI/ML integration
	deprecationDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	v1_0_0.Status = "deprecated"
	v1_0_0.DeprecationDate = &deprecationDate
	sunsetDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1_0_0.SunsetDate = &sunsetDate

	vm.supportedVersions["1.0.0"] = v1_0_0
	vm.supportedVersions["1.1.0"] = v1_1_0
	vm.supportedVersions["2.0.0"] = v2_0_0

	vm.defaultVersion = "1.1.0"
	vm.latestVersion = "2.0.0"

	// Set up compatibility map
	vm.compatibilityMap["1.0.0"] = []string{"1.0.0"}
	vm.compatibilityMap["1.1.0"] = []string{"1.0.0", "1.1.0"}
	vm.compatibilityMap["2.0.0"] = []string{"2.0.0"}
}

// AddVersion adds a new supported version
func (vm *VersionManager) AddVersion(version APIVersion) {
	vm.supportedVersions[version.String()] = version
}

// GetSupportedVersions returns all supported versions
func (vm *VersionManager) GetSupportedVersions() map[string]APIVersion {
	result := make(map[string]APIVersion)
	for k, v := range vm.supportedVersions {
		if !v.IsSunset() {
			result[k] = v
		}
	}
	return result
}

// GetLatestVersion returns the latest stable version
func (vm *VersionManager) GetLatestVersion() APIVersion {
	return vm.supportedVersions[vm.latestVersion]
}

// GetDefaultVersion returns the default version
func (vm *VersionManager) GetDefaultVersion() APIVersion {
	return vm.supportedVersions[vm.defaultVersion]
}

// ParseVersion parses a version string into an APIVersion
func (vm *VersionManager) ParseVersion(versionStr string) (APIVersion, error) {
	// Handle special cases
	switch versionStr {
	case "latest":
		return vm.GetLatestVersion(), nil
	case "default", "":
		return vm.GetDefaultVersion(), nil
	}

	// Parse semantic version
	parts := strings.Split(versionStr, ".")
	if len(parts) != 3 {
		return APIVersion{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return APIVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return APIVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return APIVersion{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	// Check if version is supported
	if version, exists := vm.supportedVersions[versionStr]; exists {
		return version, nil
	}

	// Find compatible version
	requested := APIVersion{Major: major, Minor: minor, Patch: patch}
	for _, version := range vm.supportedVersions {
		if version.IsCompatible(requested) && !version.IsSunset() {
			return version, nil
		}
	}

	return APIVersion{}, fmt.Errorf("unsupported version: %s", versionStr)
}

// ExtractVersionFromRequest extracts API version from HTTP request
func (vm *VersionManager) ExtractVersionFromRequest(c *gin.Context) (APIVersion, error) {
	// Try different methods to get version

	// 1. Header-based versioning (preferred)
	if versionHeader := c.GetHeader("API-Version"); versionHeader != "" {
		return vm.ParseVersion(versionHeader)
	}

	// 2. Accept header with custom media type
	acceptHeader := c.GetHeader("Accept")
	if acceptHeader != "" {
		// Parse Accept: application/vnd.driftmgr.v1+json
		if strings.Contains(acceptHeader, "vnd.driftmgr.v") {
			start := strings.Index(acceptHeader, "vnd.driftmgr.v") + 14
			end := strings.Index(acceptHeader[start:], "+")
			if end == -1 {
				end = strings.Index(acceptHeader[start:], ";")
			}
			if end == -1 {
				end = len(acceptHeader[start:])
			}
			versionStr := acceptHeader[start : start+end]

			// Convert v1, v2 format to semantic versioning
			if strings.HasPrefix(versionStr, "v") {
				major := versionStr[1:]
				versionStr = major + ".0.0"
			}

			return vm.ParseVersion(versionStr)
		}
	}

	// 3. Query parameter
	if versionParam := c.Query("api_version"); versionParam != "" {
		return vm.ParseVersion(versionParam)
	}

	// 4. URL path-based versioning
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/v") {
		parts := strings.Split(path[1:], "/")
		if len(parts) > 0 && strings.HasPrefix(parts[0], "v") {
			major := parts[0][1:]
			versionStr := major + ".0.0"
			return vm.ParseVersion(versionStr)
		}
	}

	// Default to default version
	return vm.GetDefaultVersion(), nil
}

// VersionMiddleware is a Gin middleware for API versioning
func (vm *VersionManager) VersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		version, err := vm.ExtractVersionFromRequest(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":              "Invalid API version",
				"message":            err.Error(),
				"supported_versions": vm.GetSupportedVersions(),
			})
			c.Abort()
			return
		}

		// Check if version is sunset
		if version.IsSunset() {
			c.JSON(http.StatusGone, gin.H{
				"error":          "API version no longer supported",
				"version":        version.String(),
				"sunset_date":    version.SunsetDate,
				"latest_version": vm.GetLatestVersion().String(),
			})
			c.Abort()
			return
		}

		// Add deprecation warning for deprecated versions
		if version.IsDeprecated() {
			c.Header("Warning", fmt.Sprintf("299 - \"API version %s is deprecated. Please migrate to %s\"",
				version.String(), vm.GetLatestVersion().String()))
			c.Header("Deprecation", "true")
			if version.SunsetDate != nil {
				c.Header("Sunset", version.SunsetDate.Format(time.RFC3339))
			}
		}

		// Store version in context for handlers
		c.Set("api_version", version)
		c.Set("version_manager", vm)

		// Add version info to response headers
		c.Header("API-Version", version.String())
		c.Header("API-Version-Status", version.Status)

		c.Next()
	}
}

// ResponseWrapper wraps responses with version-specific formatting
type ResponseWrapper struct {
	Version APIVersion    `json:"api_version"`
	Data    interface{}   `json:"data"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

// ResponseMeta contains metadata about the response
type ResponseMeta struct {
	RequestID      string           `json:"request_id,omitempty"`
	Timestamp      time.Time        `json:"timestamp"`
	ProcessingTime string           `json:"processing_time,omitempty"`
	Deprecation    *DeprecationInfo `json:"deprecation,omitempty"`
}

// DeprecationInfo contains deprecation information
type DeprecationInfo struct {
	Deprecated     bool       `json:"deprecated"`
	DeprecatedAt   *time.Time `json:"deprecated_at,omitempty"`
	SunsetDate     *time.Time `json:"sunset_date,omitempty"`
	MigrationGuide string     `json:"migration_guide,omitempty"`
}

// WrapResponse wraps a response with version-specific formatting
func (vm *VersionManager) WrapResponse(c *gin.Context, data interface{}) ResponseWrapper {
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	wrapper := ResponseWrapper{
		Version: apiVersion,
		Data:    data,
		Meta: &ResponseMeta{
			RequestID: c.GetHeader("X-Request-ID"),
			Timestamp: time.Now(),
		},
	}

	// Add deprecation info if applicable
	if apiVersion.IsDeprecated() {
		wrapper.Meta.Deprecation = &DeprecationInfo{
			Deprecated:     true,
			DeprecatedAt:   apiVersion.DeprecationDate,
			SunsetDate:     apiVersion.SunsetDate,
			MigrationGuide: fmt.Sprintf("https://docs.driftmgr.io/migration/v%d", apiVersion.Major+1),
		}
	}

	return wrapper
}

// VersionInfo represents version information endpoint response
type VersionInfo struct {
	SupportedVersions map[string]APIVersion `json:"supported_versions"`
	DefaultVersion    string                `json:"default_version"`
	LatestVersion     string                `json:"latest_version"`
	DeprecationPolicy DeprecationPolicy     `json:"deprecation_policy"`
}

// DeprecationPolicy describes the API deprecation policy
type DeprecationPolicy struct {
	NoticePeriod  string `json:"notice_period"`
	SupportPeriod string `json:"support_period"`
	Documentation string `json:"documentation"`
}

// GetVersionInfo returns version information for the /versions endpoint
func (vm *VersionManager) GetVersionInfo() VersionInfo {
	return VersionInfo{
		SupportedVersions: vm.GetSupportedVersions(),
		DefaultVersion:    vm.defaultVersion,
		LatestVersion:     vm.latestVersion,
		DeprecationPolicy: DeprecationPolicy{
			NoticePeriod:  "6 months",
			SupportPeriod: "12 months after deprecation",
			Documentation: "https://docs.driftmgr.io/api/versioning",
		},
	}
}

// MigrationHelper provides migration assistance between versions
type MigrationHelper struct {
	vm *VersionManager
}

// NewMigrationHelper creates a new migration helper
func NewMigrationHelper(vm *VersionManager) *MigrationHelper {
	return &MigrationHelper{vm: vm}
}

// TransformResponse transforms a response from one version to another
func (mh *MigrationHelper) TransformResponse(ctx context.Context, data interface{}, fromVersion, toVersion APIVersion) (interface{}, error) {
	// Implement version-specific transformations
	switch {
	case fromVersion.Major == 1 && toVersion.Major == 2:
		return mh.transformV1ToV2(data)
	case fromVersion.Major == 2 && toVersion.Major == 1:
		return mh.transformV2ToV1(data)
	default:
		return data, nil // No transformation needed
	}
}

// transformV1ToV2 transforms v1 response to v2 format
func (mh *MigrationHelper) transformV1ToV2(data interface{}) (interface{}, error) {
	// Add v2-specific fields, restructure data, etc.
	if dataMap, ok := data.(map[string]interface{}); ok {
		// Add v2 metadata
		dataMap["_metadata"] = map[string]interface{}{
			"format_version": "2.0",
			"migrated_from":  "1.x",
		}

		// Transform field names if needed
		if resources, exists := dataMap["resources"]; exists {
			if resourceList, ok := resources.([]interface{}); ok {
				for _, resource := range resourceList {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						// Add v2 resource fields
						resourceMap["resource_version"] = "2.0"
					}
				}
			}
		}
	}

	return data, nil
}

// transformV2ToV1 transforms v2 response to v1 format
func (mh *MigrationHelper) transformV2ToV1(data interface{}) (interface{}, error) {
	// Remove v2-specific fields, restructure data for v1 compatibility
	if dataMap, ok := data.(map[string]interface{}); ok {
		// Remove v2-specific metadata
		delete(dataMap, "_metadata")

		// Transform field names back to v1 format
		if resources, exists := dataMap["resources"]; exists {
			if resourceList, ok := resources.([]interface{}); ok {
				for _, resource := range resourceList {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						// Remove v2-specific fields
						delete(resourceMap, "resource_version")
					}
				}
			}
		}
	}

	return data, nil
}

// CompatibilityChecker checks compatibility between versions
type CompatibilityChecker struct {
	vm *VersionManager
}

// NewCompatibilityChecker creates a new compatibility checker
func NewCompatibilityChecker(vm *VersionManager) *CompatibilityChecker {
	return &CompatibilityChecker{vm: vm}
}

// CheckCompatibility checks if a request is compatible with the target version
func (cc *CompatibilityChecker) CheckCompatibility(requestData interface{}, targetVersion APIVersion) (bool, []string) {
	var issues []string

	// Check for version-specific field compatibility
	if dataMap, ok := requestData.(map[string]interface{}); ok {
		switch targetVersion.Major {
		case 1:
			issues = append(issues, cc.checkV1Compatibility(dataMap)...)
		case 2:
			issues = append(issues, cc.checkV2Compatibility(dataMap)...)
		}
	}

	return len(issues) == 0, issues
}

// checkV1Compatibility checks v1-specific compatibility issues
func (cc *CompatibilityChecker) checkV1Compatibility(data map[string]interface{}) []string {
	var issues []string

	// Check for fields that don't exist in v1
	v2OnlyFields := []string{"_metadata", "resource_version", "streaming_support"}
	for _, field := range v2OnlyFields {
		if _, exists := data[field]; exists {
			issues = append(issues, fmt.Sprintf("Field '%s' is not supported in API v1", field))
		}
	}

	return issues
}

// checkV2Compatibility checks v2-specific compatibility issues
func (cc *CompatibilityChecker) checkV2Compatibility(data map[string]interface{}) []string {
	var issues []string

	// Check for deprecated v1 fields
	deprecatedFields := []string{"legacy_format", "v1_compatibility"}
	for _, field := range deprecatedFields {
		if _, exists := data[field]; exists {
			issues = append(issues, fmt.Sprintf("Field '%s' is deprecated in API v2", field))
		}
	}

	return issues
}
