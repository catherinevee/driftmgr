package discovery

// ValidationResult represents validation results
type ValidationResult struct {
	Region        string   `json:"region"`
	DriftmgrCount int      `json:"driftmgr_count"`
	CLICount      int      `json:"cli_count"`
	Match         bool     `json:"match"`
	MissingInCLI  []string `json:"missing_in_cli,omitempty"`
	MissingInDriftmgr []string `json:"missing_in_driftmgr,omitempty"`
}

// ScanOptions represents scan options
type ScanOptions struct {
	Path        string
	Recursive   bool
	Pattern     string
	MaxDepth    int
	Workers     int
	FilterTypes []string
}

// BackendScanner interface for backend scanning
type BackendScanner interface {
	Scan(ctx interface{}, opts ScanOptions) ([]*BackendConfig, error)
}

// NewBackendScanner creates a new backend scanner
func NewBackendScanner() BackendScanner {
	return &backendScanner{}
}

type backendScanner struct{}

func (s *backendScanner) Scan(ctx interface{}, opts ScanOptions) ([]*BackendConfig, error) {
	// Simplified implementation
	return []*BackendConfig{}, nil
}