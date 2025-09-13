package testutils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// MockServerConfig holds configuration for a mock HTTP server.
type MockServerConfig struct {
	Method       string
	Path         string
	Status       int
	ResponseBody interface{}
	Headers     map[string]string
}

// SetupMockServer creates a test HTTP server with the given configuration.
// Returns the server instance and its base URL.
func SetupMockServer(t *testing.T, configs []MockServerConfig) (*httptest.Server, string) {
	handler := http.NewServeMux()

	for _, cfg := range configs {
		cfg := cfg // Create a new variable for the closure
		handler.HandleFunc(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
			// Verify the request method
			if r.Method != cfg.Method {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			// Set response headers
			for key, value := range cfg.Headers {
				w.Header().Set(key, value)
			}

			// Set status code
			w.WriteHeader(cfg.Status)

			// Write response body if provided
			if cfg.ResponseBody != nil {
				switch body := cfg.ResponseBody.(type) {
				case []byte:
					_, _ = w.Write(body)
				case string:
					_, _ = w.Write([]byte(body))
				default:
					err := json.NewEncoder(w).Encode(body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		})
	}

	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})

	return server, server.URL
}

// CreateMockJSONResponse creates a mock HTTP response with JSON content.
func CreateMockJSONResponse(t *testing.T, statusCode int, body interface{}) *http.Response {
	jsonData, err := json.Marshal(body)
	require.NoError(t, err)

	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: &mockReadCloser{data: jsonData},
	}
}

type mockReadCloser struct {
	data []byte
	read int
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.read >= len(m.data) {
		return 0, nil // EOF
	}
	n = copy(p, m.data[m.read:])
	m.read += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}
