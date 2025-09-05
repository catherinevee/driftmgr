package compliance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// OPAEngine provides policy evaluation using Open Policy Agent
type OPAEngine struct {
	endpoint       string
	httpClient     *http.Client
	policies       map[string]*Policy
	cacheDuration  time.Duration
	cache          map[string]*CachedDecision
	mu             sync.RWMutex
	pluginMode     bool
	localPolicies  string
}

// Policy represents an OPA policy
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Package     string                 `json:"package"`
	Rules       string                 `json:"rules"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PolicyInput represents input for policy evaluation
type PolicyInput struct {
	Resource   interface{}            `json:"resource"`
	Action     string                 `json:"action"`
	Principal  string                 `json:"principal,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Provider   string                 `json:"provider,omitempty"`
	Region     string                 `json:"region,omitempty"`
	Tags       map[string]string      `json:"tags,omitempty"`
}

// PolicyDecision represents the policy evaluation result
type PolicyDecision struct {
	Allow       bool                   `json:"allow"`
	Reasons     []string               `json:"reasons,omitempty"`
	Violations  []PolicyViolation      `json:"violations,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	Rule        string                 `json:"rule"`
	Message     string                 `json:"message"`
	Severity    string                 `json:"severity"`
	Resource    string                 `json:"resource,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
}

// CachedDecision represents a cached policy decision
type CachedDecision struct {
	Decision  *PolicyDecision
	ExpiresAt time.Time
}

// OPAConfig configures the OPA engine
type OPAConfig struct {
	Endpoint       string        // OPA server endpoint (e.g., http://localhost:8181)
	PluginMode     bool          // Use OPA as external plugin vs embedded
	LocalPolicies  string        // Path to local policy files
	CacheDuration  time.Duration // Cache duration for decisions
	Timeout        time.Duration // HTTP timeout for OPA calls
}

// NewOPAEngine creates a new OPA policy engine
func NewOPAEngine(config OPAConfig) *OPAEngine {
	if config.CacheDuration == 0 {
		config.CacheDuration = 5 * time.Minute
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	
	return &OPAEngine{
		endpoint:      config.Endpoint,
		pluginMode:    config.PluginMode,
		localPolicies: config.LocalPolicies,
		cacheDuration: config.CacheDuration,
		policies:      make(map[string]*Policy),
		cache:         make(map[string]*CachedDecision),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// LoadPolicies loads policies from local files or OPA server
func (e *OPAEngine) LoadPolicies(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.localPolicies != "" {
		return e.loadLocalPolicies()
	}
	
	if e.pluginMode && e.endpoint != "" {
		return e.loadRemotePolicies(ctx)
	}
	
	return nil
}

// loadLocalPolicies loads policies from local filesystem
func (e *OPAEngine) loadLocalPolicies() error {
	policyFiles, err := filepath.Glob(filepath.Join(e.localPolicies, "*.rego"))
	if err != nil {
		return fmt.Errorf("failed to list policy files: %w", err)
	}
	
	for _, file := range policyFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read policy %s: %w", file, err)
		}
		
		policy := &Policy{
			ID:        filepath.Base(file),
			Name:      strings.TrimSuffix(filepath.Base(file), ".rego"),
			Rules:     string(content),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		// Extract package name from content
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "package ") {
				policy.Package = strings.TrimPrefix(strings.TrimSpace(line), "package ")
				break
			}
		}
		
		e.policies[policy.ID] = policy
	}
	
	return nil
}

// loadRemotePolicies loads policies from OPA server
func (e *OPAEngine) loadRemotePolicies(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", e.endpoint+"/v1/policies", nil)
	if err != nil {
		return err
	}
	
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch policies: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, body)
	}
	
	var policies map[string]*Policy
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return fmt.Errorf("failed to decode policies: %w", err)
	}
	
	e.policies = policies
	return nil
}

// Evaluate evaluates a policy against input
func (e *OPAEngine) Evaluate(ctx context.Context, policyPackage string, input PolicyInput) (*PolicyDecision, error) {
	// Check cache first
	cacheKey := e.getCacheKey(policyPackage, input)
	if cached := e.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}
	
	var decision *PolicyDecision
	var err error
	
	if e.pluginMode && e.endpoint != "" {
		decision, err = e.evaluateRemote(ctx, policyPackage, input)
	} else {
		decision, err = e.evaluateLocal(ctx, policyPackage, input)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Cache the decision
	e.putInCache(cacheKey, decision)
	
	return decision, nil
}

// evaluateRemote evaluates policy using OPA server
func (e *OPAEngine) evaluateRemote(ctx context.Context, policyPackage string, input PolicyInput) (*PolicyDecision, error) {
	url := fmt.Sprintf("%s/v1/data/%s", e.endpoint, strings.ReplaceAll(policyPackage, ".", "/"))
	
	body, err := json.Marshal(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, body)
	}
	
	var result struct {
		Result PolicyDecision `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}
	
	result.Result.EvaluatedAt = time.Now()
	return &result.Result, nil
}

// evaluateLocal evaluates policy locally (stub for demonstration)
func (e *OPAEngine) evaluateLocal(ctx context.Context, policyPackage string, input PolicyInput) (*PolicyDecision, error) {
	// In a real implementation, this would use OPA's Go API
	// For now, return a mock decision based on simple rules
	
	decision := &PolicyDecision{
		Allow:       true,
		Reasons:     []string{},
		Violations:  []PolicyViolation{},
		Suggestions: []string{},
		EvaluatedAt: time.Now(),
	}
	
	// Example validation rules
	if input.Provider == "aws" && input.Action == "delete" {
		if input.Tags != nil && input.Tags["Environment"] == "production" {
			decision.Allow = false
			decision.Violations = append(decision.Violations, PolicyViolation{
				Rule:        "no_delete_production",
				Message:     "Cannot delete production resources",
				Severity:    "high",
				Resource:    fmt.Sprintf("%v", input.Resource),
				Remediation: "Remove production tag or use staging environment",
			})
		}
	}
	
	// Check for required tags
	requiredTags := []string{"Owner", "Environment", "CostCenter"}
	for _, tag := range requiredTags {
		if input.Tags == nil || input.Tags[tag] == "" {
			decision.Violations = append(decision.Violations, PolicyViolation{
				Rule:     "required_tags",
				Message:  fmt.Sprintf("Missing required tag: %s", tag),
				Severity: "medium",
				Resource: fmt.Sprintf("%v", input.Resource),
				Remediation: fmt.Sprintf("Add %s tag to the resource", tag),
			})
		}
	}
	
	if len(decision.Violations) > 0 {
		decision.Allow = false
	}
	
	return decision, nil
}

// UploadPolicy uploads a new policy to OPA
func (e *OPAEngine) UploadPolicy(ctx context.Context, policy *Policy) error {
	if !e.pluginMode || e.endpoint == "" {
		// Save locally
		e.mu.Lock()
		e.policies[policy.ID] = policy
		e.mu.Unlock()
		
		if e.localPolicies != "" {
			filename := filepath.Join(e.localPolicies, policy.ID+".rego")
			return os.WriteFile(filename, []byte(policy.Rules), 0644)
		}
		return nil
	}
	
	// Upload to OPA server
	url := fmt.Sprintf("%s/v1/configs/policies/%s", e.endpoint, policy.ID)
	
	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(policy.Rules))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload policy: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, body)
	}
	
	e.mu.Lock()
	e.policies[policy.ID] = policy
	e.mu.Unlock()
	
	return nil
}

// DeletePolicy deletes a policy
func (e *OPAEngine) DeletePolicy(ctx context.Context, policyID string) error {
	e.mu.Lock()
	delete(e.policies, policyID)
	e.mu.Unlock()
	
	if e.localPolicies != "" {
		filename := filepath.Join(e.localPolicies, policyID+".rego")
		return os.Remove(filename)
	}
	
	if e.pluginMode && e.endpoint != "" {
		url := fmt.Sprintf("%s/v1/configs/policies/%s", e.endpoint, policyID)
		req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
		if err != nil {
			return err
		}
		
		resp, err := e.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete policy: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, body)
		}
	}
	
	return nil
}

// GetPolicy retrieves a policy by ID
func (e *OPAEngine) GetPolicy(policyID string) (*Policy, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	policy, exists := e.policies[policyID]
	return policy, exists
}

// ListPolicies returns all loaded policies
func (e *OPAEngine) ListPolicies() []*Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	policies := make([]*Policy, 0, len(e.policies))
	for _, policy := range e.policies {
		policies = append(policies, policy)
	}
	return policies
}

// Cache management

func (e *OPAEngine) getCacheKey(policyPackage string, input PolicyInput) string {
	data, _ := json.Marshal(input)
	return fmt.Sprintf("%s:%x", policyPackage, data)
}

func (e *OPAEngine) getFromCache(key string) *PolicyDecision {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	cached, exists := e.cache[key]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil
	}
	
	return cached.Decision
}

func (e *OPAEngine) putInCache(key string, decision *PolicyDecision) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.cache[key] = &CachedDecision{
		Decision:  decision,
		ExpiresAt: time.Now().Add(e.cacheDuration),
	}
}

// ClearCache clears the decision cache
func (e *OPAEngine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.cache = make(map[string]*CachedDecision)
}