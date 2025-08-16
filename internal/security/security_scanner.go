package security

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// SecurityScanner provides security assessment and compliance checking
type SecurityScanner struct {
	providers map[string]SecurityProvider
}

// SecurityProvider interface for different cloud providers
type SecurityProvider interface {
	AssessResource(ctx context.Context, resource models.Resource) (*SecurityAssessment, error)
	CheckCompliance(ctx context.Context, resource models.Resource, framework string) (*ComplianceResult, error)
}

// SecurityAssessment represents security assessment results
type SecurityAssessment struct {
	ResourceID      string
	OverallScore    int
	RiskFactors     []RiskFactor
	ComplianceGaps  []ComplianceGap
	Recommendations []SecurityRecommendation
	LastAssessment  time.Time
}

// RiskFactor represents a security risk
type RiskFactor struct {
	Category    string
	Description string
	Severity    string // "low", "medium", "high", "critical"
	Impact      string
	Remediation string
}

// ComplianceGap represents a compliance gap
type ComplianceGap struct {
	Framework   string
	Category    string
	Description string
	Severity    string
	Remediation string
}

// SecurityRecommendation represents a security recommendation
type SecurityRecommendation struct {
	Category    string
	Description string
	Priority    string // "low", "medium", "high", "critical"
	Effort      string // "easy", "medium", "hard"
	Impact      string
}

// ComplianceResult represents compliance checking results
type ComplianceResult struct {
	Framework      string
	OverallScore   int
	Compliant      bool
	Violations     []ComplianceViolation
	LastChecked    time.Time
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Rule        string
	Description string
	Severity    string
	Remediation string
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		providers: make(map[string]SecurityProvider),
	}
}

// RegisterProvider registers a security provider
func (ss *SecurityScanner) RegisterProvider(name string, provider SecurityProvider) {
	ss.providers[name] = provider
}

// AssessResource performs security assessment on a resource
func (ss *SecurityScanner) AssessResource(ctx context.Context, resource models.Resource) (*SecurityAssessment, error) {
	provider, exists := ss.providers[resource.Provider]
	if !exists {
		return nil, fmt.Errorf("security provider for %s not registered", resource.Provider)
	}

	return provider.AssessResource(ctx, resource)
}

// CheckCompliance checks compliance for a specific framework
func (ss *SecurityScanner) CheckCompliance(ctx context.Context, resource models.Resource, framework string) (*ComplianceResult, error) {
	provider, exists := ss.providers[resource.Provider]
	if !exists {
		return nil, fmt.Errorf("security provider for %s not registered", resource.Provider)
	}

	return provider.CheckCompliance(ctx, resource, framework)
}
