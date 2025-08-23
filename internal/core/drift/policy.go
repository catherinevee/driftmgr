package drift

import (
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// PolicyEngine evaluates drift against policies
type PolicyEngine struct {
	policies []Policy
}

// Policy defines a drift policy
type Policy struct {
	Name        string
	Environment string
	Rules       []Rule
}

// Rule defines a policy rule
type Rule struct {
	ResourceType string
	DriftType    string
	Severity     string
	Action       string
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		policies: loadDefaultPolicies(),
	}
}

// EvaluateDrifts evaluates drifts against policies
func (pe *PolicyEngine) EvaluateDrifts(drifts []models.DriftItem, environment string) []models.DriftItem {
	evaluated := make([]models.DriftItem, 0)

	for _, drift := range drifts {
		if pe.shouldInclude(drift, environment) {
			evaluated = append(evaluated, drift)
		}
	}

	return evaluated
}

func (pe *PolicyEngine) shouldInclude(drift models.DriftItem, environment string) bool {
	// Apply policy rules
	for _, policy := range pe.policies {
		if policy.Environment == environment || policy.Environment == "*" {
			for _, rule := range policy.Rules {
				if pe.matchesRule(drift, rule) {
					return rule.Action == "include"
				}
			}
		}
	}
	return true
}

func (pe *PolicyEngine) matchesRule(drift models.DriftItem, rule Rule) bool {
	if rule.ResourceType != "*" && rule.ResourceType != drift.ResourceType {
		return false
	}
	if rule.DriftType != "*" && rule.DriftType != drift.DriftType {
		return false
	}
	if rule.Severity != "*" && rule.Severity != drift.Severity {
		return false
	}
	return true
}

func loadDefaultPolicies() []Policy {
	return []Policy{
		{
			Name:        "production_critical",
			Environment: "production",
			Rules: []Rule{
				{ResourceType: "*", DriftType: "*", Severity: "critical", Action: "include"},
				{ResourceType: "*", DriftType: "*", Severity: "high", Action: "include"},
			},
		},
		{
			Name:        "development_filter",
			Environment: "development",
			Rules: []Rule{
				{ResourceType: "*", DriftType: "added", Severity: "low", Action: "exclude"},
			},
		},
	}
}
