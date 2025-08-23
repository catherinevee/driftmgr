package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		v := NewValidator(nil)
		assert.NotNil(t, v)
		assert.NotNil(t, v.config)
		assert.Equal(t, 10000, v.config.MaxStringLength)
	})
	
	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			MaxStringLength: 5000,
			StrictMode:      false,
		}
		v := NewValidator(config)
		assert.NotNil(t, v)
		assert.Equal(t, 5000, v.config.MaxStringLength)
		assert.False(t, v.config.StrictMode)
	})
}

func TestValidateString(t *testing.T) {
	v := NewValidator(&Config{
		MaxStringLength: 100,
		StrictMode:      true,
		AllowHTML:       false,
	})
	
	tests := []struct {
		name      string
		input     string
		fieldName string
		wantErr   bool
		expected  string
	}{
		{
			name:      "valid string",
			input:     "hello world",
			fieldName: "test",
			wantErr:   false,
			expected:  "hello world",
		},
		{
			name:      "string with HTML",
			input:     "<script>alert('xss')</script>",
			fieldName: "test",
			wantErr:   true,
			expected:  "",
		},
		{
			name:      "string too long",
			input:     strings.Repeat("a", 101),
			fieldName: "test",
			wantErr:   true,
			expected:  "",
		},
		{
			name:      "string with null bytes",
			input:     "hello\x00world",
			fieldName: "test",
			wantErr:   false,
			expected:  "helloworld",
		},
		{
			name:      "SQL injection attempt",
			input:     "'; DROP TABLE users; --",
			fieldName: "test",
			wantErr:   true,
			expected:  "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.ValidateString(tt.input, tt.fieldName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	v := NewValidator(nil)
	
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "test@mail.example.com", false},
		{"valid email with plus", "test+tag@example.com", false},
		{"invalid - no @", "testexample.com", true},
		{"invalid - no domain", "test@", true},
		{"invalid - no local part", "@example.com", true},
		{"invalid - multiple @", "test@@example.com", true},
		{"invalid - spaces", "test @example.com", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	v := NewValidator(&Config{StrictMode: true})
	
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid HTTP URL", "http://example.com", false},
		{"valid HTTPS URL", "https://example.com", false},
		{"valid URL with path", "https://example.com/path/to/resource", false},
		{"valid URL with query", "https://example.com?key=value", false},
		{"invalid - no scheme", "example.com", true},
		{"invalid - invalid scheme", "ftp://example.com", true},
		{"invalid - localhost in strict mode", "http://localhost:8080", true},
		{"invalid - 127.0.0.1 in strict mode", "http://127.0.0.1", true},
		{"invalid - no host", "http://", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	v := NewValidator(&Config{StrictMode: true})
	
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv4", "8.8.8.8", false},
		{"valid IPv6", "2001:4860:4860::8888", false},
		{"invalid - private IPv4 in strict", "192.168.1.1", true},
		{"invalid - loopback in strict", "127.0.0.1", true},
		{"invalid - not an IP", "not.an.ip", true},
		{"invalid - empty", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateIP(tt.ip)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	v := NewValidator(&Config{
		MaxJSONDepth:    3,
		MaxStringLength: 1000,
	})
	
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid simple JSON",
			json:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "valid nested JSON",
			json:    `{"level1": {"level2": {"level3": "value"}}}`,
			wantErr: false,
		},
		{
			name:    "invalid - too deep",
			json:    `{"l1": {"l2": {"l3": {"l4": "value"}}}}`,
			wantErr: true,
		},
		{
			name:    "invalid - not JSON",
			json:    `not json`,
			wantErr: true,
		},
		{
			name:    "invalid - too large",
			json:    `{"key": "` + strings.Repeat("a", 1001) + `"}`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.ValidateJSON(tt.json)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestValidateCloudResourceID(t *testing.T) {
	v := NewValidator(nil)
	
	tests := []struct {
		name       string
		resourceID string
		provider   string
		wantErr    bool
	}{
		// AWS tests
		{
			name:       "valid AWS ARN",
			resourceID: "arn:aws:ec2:us-west-2:123456789012:instance/i-1234567890abcdef0",
			provider:   "aws",
			wantErr:    false,
		},
		{
			name:       "valid AWS resource ID",
			resourceID: "i-1234567890abcdef0",
			provider:   "aws",
			wantErr:    false,
		},
		{
			name:       "invalid AWS ARN",
			resourceID: "arn:invalid",
			provider:   "aws",
			wantErr:    true,
		},
		// Azure tests
		{
			name:       "valid Azure resource ID",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.Compute/virtualMachines/myVM",
			provider:   "azure",
			wantErr:    false,
		},
		{
			name:       "invalid Azure resource ID",
			resourceID: "/invalid/azure/id",
			provider:   "azure",
			wantErr:    true,
		},
		// GCP tests
		{
			name:       "valid GCP resource ID",
			resourceID: "projects/my-project/zones/us-central1-a/instances/my-instance",
			provider:   "gcp",
			wantErr:    false,
		},
		{
			name:       "invalid GCP resource ID",
			resourceID: "invalid/gcp/id",
			provider:   "gcp",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCloudResourceID(tt.resourceID, tt.provider)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	v := NewValidator(&Config{StrictMode: true})
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "/home/user/file.txt", false},
		{"valid Windows path", "C:\\Users\\user\\file.txt", false},
		{"invalid - path traversal", "../../../etc/passwd", true},
		{"invalid - null bytes", "/home/user\x00/file.txt", true},
		{"invalid - command injection chars", "/home/user; rm -rf /", true},
		{"invalid - too long", strings.Repeat("a", 4097), true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateFilePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	v := NewValidator(nil)
	
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{"valid command", "ls -la", true}, // Has shell chars
		{"valid simple command", "terraform apply", false},
		{"invalid - rm -rf", "rm -rf /", true},
		{"invalid - drop table", "DROP TABLE users", true},
		{"invalid - command injection", "ls; cat /etc/passwd", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCommand(tt.cmd)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	v := NewValidator(&Config{
		AllowHTML:    false,
		AllowScripts: false,
	})
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "HTML escaped",
			input:    "<div>hello</div>",
			expected: "&lt;div&gt;hello&lt;/div&gt;",
		},
		{
			name:     "null bytes removed",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "non-printable removed",
			input:    "hello\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "whitespace trimmed",
			input:    "  hello world  ",
			expected: "hello world",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.sanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetJSONDepth(t *testing.T) {
	v := NewValidator(nil)
	
	tests := []struct {
		name     string
		data     interface{}
		expected int
	}{
		{
			name:     "flat object",
			data:     map[string]interface{}{"key": "value"},
			expected: 1,
		},
		{
			name: "nested object",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "value",
				},
			},
			expected: 2,
		},
		{
			name: "array with objects",
			data: []interface{}{
				map[string]interface{}{"key": "value"},
			},
			expected: 2,
		},
		{
			name:     "primitive",
			data:     "string",
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depth := v.getJSONDepth(tt.data)
			assert.Equal(t, tt.expected, depth)
		})
	}
}

func BenchmarkValidateString(b *testing.B) {
	v := NewValidator(nil)
	input := "This is a test string with some content"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.ValidateString(input, "benchmark")
	}
}

func BenchmarkValidateJSON(b *testing.B) {
	v := NewValidator(nil)
	json := `{"key": "value", "nested": {"field": "data"}}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.ValidateJSON(json)
	}
}

func BenchmarkSanitizeString(b *testing.B) {
	v := NewValidator(nil)
	input := "<script>alert('test')</script>Hello World!"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.sanitizeString(input)
	}
}