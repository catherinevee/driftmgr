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

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
)

// OPAEngine provides policy evaluation using Open Policy Agent
type OPAEngine struct {
	endpoint      string
	httpClient    *http.Client
	policies      map[string]*Policy
	cacheDuration time.Duration
	cache         map[string]*CachedDecision
	mu            sync.RWMutex
	pluginMode    bool
	localPolicies string
	compiler      *ast.Compiler
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
	Resource  interface{}            `json:"resource"`
	Action    string                 `json:"action"`
	Principal string                 `json:"principal,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	Region    string                 `json:"region,omitempty"`
	Tags      map[string]string      `json:"tags,omitempty"`
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
	Endpoint      string        // OPA server endpoint (e.g., http://localhost:8181)
	PluginMode    bool          // Use OPA as external plugin vs embedded
	LocalPolicies string        // Path to local policy files
	CacheDuration time.Duration // Cache duration for decisions
	Timeout       time.Duration // HTTP timeout for OPA calls
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
		compiler:      ast.NewCompiler(),
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

	// Collect all policy modules for compilation
	modules := make(map[string]*ast.Module)

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

		// Parse the policy module
		module, err := ast.ParseModule(file, string(content))
		if err != nil {
			return fmt.Errorf("failed to parse policy %s: %w", file, err)
		}

		// Extract package name from module (remove "data." prefix if present)
		packageName := module.Package.Path.String()
		if strings.HasPrefix(packageName, "data.") {
			packageName = strings.TrimPrefix(packageName, "data.")
		}
		policy.Package = packageName

		// Store the module for compilation
		modules[file] = module
		e.policies[policy.ID] = policy
	}

	// Compile all modules
	e.compiler.Compile(modules)
	if e.compiler.Failed() {
		return fmt.Errorf("failed to compile policies: %v", e.compiler.Errors)
	}

	// Debug logging (commented out for production)
	// fmt.Printf("DEBUG: Loaded %d policies\n", len(e.policies))
	// for id, policy := range e.policies {
	//     fmt.Printf("DEBUG: Policy %s: Package=%s, Name=%s\n", id, policy.Package, policy.Name)
	// }

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

// evaluateLocal evaluates policy locally using OPA Go SDK
func (e *OPAEngine) evaluateLocal(ctx context.Context, policyPackage string, input PolicyInput) (*PolicyDecision, error) {
	// Find a policy with matching package
	var policy *Policy
	for _, p := range e.policies {
		if p.Package == policyPackage {
			policy = p
			break
		}
	}

	if policy == nil {
		return nil, fmt.Errorf("no policy found for package: %s", policyPackage)
	}

	// Convert input to map for OPA
	inputMap := map[string]interface{}{
		"resource":  input.Resource,
		"action":    input.Action,
		"principal": input.Principal,
		"context":   input.Context,
		"provider":  input.Provider,
		"region":    input.Region,
		"tags":      input.Tags,
	}

	// Create a query that returns both allow and violations
	query := fmt.Sprintf("data.%s", strings.ReplaceAll(policyPackage, ".", "."))
	results, err := rego.New(
		rego.Query(query),
		rego.Compiler(e.compiler),
		rego.Input(inputMap),
	).Eval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Debug logging (commented out for production)
	// fmt.Printf("DEBUG: Query: %s\n", query)
	// fmt.Printf("DEBUG: Input: %+v\n", inputMap)
	// fmt.Printf("DEBUG: Results: %+v\n", results)

	// Process results
	decision := &PolicyDecision{
		Allow:       true,
		Reasons:     []string{},
		Violations:  []PolicyViolation{},
		Suggestions: []string{},
		EvaluatedAt: time.Now(),
	}

	// Parse OPA results
	for _, result := range results {
		for _, expression := range result.Expressions {
			if expression.Value != nil {
				// Handle the full data structure
				if dataMap, ok := expression.Value.(map[string]interface{}); ok {
					// Extract allow decision
					if allow, ok := dataMap["allow"].(bool); ok {
						decision.Allow = allow
					}

					// Extract violations
					if violations, ok := dataMap["violations"].(map[string]interface{}); ok {
						for key, violation := range violations {
							if violationMap, ok := violation.(map[string]interface{}); ok {
								// Handle violation objects
								pv := PolicyViolation{
									Rule:        getString(violationMap, "rule"),
									Message:     getString(violationMap, "message"),
									Severity:    getString(violationMap, "severity"),
									Resource:    getString(violationMap, "resource"),
									Remediation: getString(violationMap, "remediation"),
								}
								decision.Violations = append(decision.Violations, pv)
							} else {
								// Handle violation keys (when violation is just a string key)
								pv := PolicyViolation{
									Rule:        "required_tags",
									Message:     fmt.Sprintf("Policy violation: %s", key),
									Severity:    "medium",
									Resource:    fmt.Sprintf("%v", input.Resource),
									Remediation: "Fix the policy violation",
								}
								decision.Violations = append(decision.Violations, pv)
							}
						}
					} else if violations, ok := dataMap["violations"].([]interface{}); ok {
						// Handle violations as an array of strings
						for _, violation := range violations {
							if violationStr, ok := violation.(string); ok {
								pv := PolicyViolation{
									Rule:        "required_tags",
									Message:     fmt.Sprintf("Policy violation: %s", violationStr),
									Severity:    "medium",
									Resource:    fmt.Sprintf("%v", input.Resource),
									Remediation: "Fix the policy violation",
								}
								decision.Violations = append(decision.Violations, pv)
							}
						}
					}
				}
			}
		}
	}

	// If there are violations, deny access
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
	defer e.mu.Unlock()

	if e.localPolicies != "" {
		filename := filepath.Join(e.localPolicies, policyID+".rego")
		err := os.Remove(filename)
		if err != nil {
			return err
		}
		// Delete from in-memory map using the full filename
		delete(e.policies, policyID+".rego")
		return nil
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

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
