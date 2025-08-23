package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecureConfig(t *testing.T) {
	// Clean up any existing encryption key
	defer os.Remove(".encryption.key")
	
	t.Run("creates config successfully", func(t *testing.T) {
		sc, err := NewSecureConfig()
		require.NoError(t, err)
		assert.NotNil(t, sc)
		assert.NotNil(t, sc.encryptionKey)
		assert.NotNil(t, sc.auditLogger)
	})
	
	t.Run("loads environment variables", func(t *testing.T) {
		os.Setenv("DRIFTMGR_TEST_VAR", "test_value")
		defer os.Unsetenv("DRIFTMGR_TEST_VAR")
		
		sc, err := NewSecureConfig()
		require.NoError(t, err)
		
		value, err := sc.GetString("TEST_VAR")
		assert.NoError(t, err)
		assert.Equal(t, "test_value", value)
	})
}

func TestSecureConfigGetString(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("gets existing value", func(t *testing.T) {
		os.Setenv("DRIFTMGR_TEST_KEY", "test_value")
		defer os.Unsetenv("DRIFTMGR_TEST_KEY")
		
		sc.loadEnvironmentVariables()
		
		value, err := sc.GetString("TEST_KEY")
		assert.NoError(t, err)
		assert.Equal(t, "test_value", value)
	})
	
	t.Run("returns error for non-existent key", func(t *testing.T) {
		value, err := sc.GetString("NON_EXISTENT_KEY")
		assert.Error(t, err)
		assert.Empty(t, value)
	})
}

func TestSecureConfigSetSecure(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("sets and encrypts value", func(t *testing.T) {
		err := sc.SetSecure("secure_key", "sensitive_value")
		assert.NoError(t, err)
		
		// Retrieve the value
		value, err := sc.GetSecureString("secure_key")
		assert.NoError(t, err)
		assert.Equal(t, "sensitive_value", value)
	})
	
	t.Run("validates value with registered validator", func(t *testing.T) {
		sc.RegisterValidator("validated_key", func(value interface{}) error {
			str, ok := value.(string)
			if !ok || len(str) < 5 {
				return assert.AnError
			}
			return nil
		})
		
		// Should fail validation
		err := sc.SetSecure("validated_key", "abc")
		assert.Error(t, err)
		
		// Should pass validation
		err = sc.SetSecure("validated_key", "valid_value")
		assert.NoError(t, err)
	})
}

func TestSecureConfigEncryption(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("encrypts and decrypts correctly", func(t *testing.T) {
		plaintext := "secret_data"
		
		encrypted, err := sc.encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, encrypted)
		
		decrypted, err := sc.decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})
	
	t.Run("produces different ciphertext for same plaintext", func(t *testing.T) {
		plaintext := "secret_data"
		
		encrypted1, err := sc.encrypt(plaintext)
		require.NoError(t, err)
		
		encrypted2, err := sc.encrypt(plaintext)
		require.NoError(t, err)
		
		// Should be different due to random nonce
		assert.NotEqual(t, encrypted1, encrypted2)
		
		// But both should decrypt to same value
		decrypted1, err := sc.decrypt(encrypted1)
		require.NoError(t, err)
		
		decrypted2, err := sc.decrypt(encrypted2)
		require.NoError(t, err)
		
		assert.Equal(t, decrypted1, decrypted2)
	})
}

func TestSecureConfigValidation(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("validates all registered validators", func(t *testing.T) {
		// Register validators
		sc.RegisterValidator("port", func(value interface{}) error {
			port, ok := value.(string)
			if !ok || port < "1" || port > "65535" {
				return assert.AnError
			}
			return nil
		})
		
		// Set valid value
		sc.configFile["port"] = "8080"
		
		err := sc.Validate()
		assert.NoError(t, err)
		
		// Set invalid value
		sc.configFile["port"] = "99999"
		
		err = sc.Validate()
		assert.Error(t, err)
	})
}

func TestSecureConfigCaching(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("caches values", func(t *testing.T) {
		// Set a value
		sc.configFile["cached_key"] = "cached_value"
		
		// First access should cache
		value1, err := sc.get("cached_key")
		assert.NoError(t, err)
		assert.Equal(t, "cached_value", value1)
		
		// Modify underlying value
		sc.configFile["cached_key"] = "modified_value"
		
		// Should still return cached value
		value2, err := sc.get("cached_key")
		assert.NoError(t, err)
		assert.Equal(t, "cached_value", value2)
		
		// Invalidate cache
		sc.invalidateCache("cached_key")
		
		// Should return new value
		value3, err := sc.get("cached_key")
		assert.NoError(t, err)
		assert.Equal(t, "modified_value", value3)
	})
}

func TestSecureConfigCloudCredentials(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("AWS credentials", func(t *testing.T) {
		os.Setenv("DRIFTMGR_AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
		os.Setenv("DRIFTMGR_AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
		os.Setenv("DRIFTMGR_AWS_REGION", "us-west-2")
		defer func() {
			os.Unsetenv("DRIFTMGR_AWS_ACCESS_KEY_ID")
			os.Unsetenv("DRIFTMGR_AWS_SECRET_ACCESS_KEY")
			os.Unsetenv("DRIFTMGR_AWS_REGION")
		}()
		
		sc.loadEnvironmentVariables()
		
		creds, err := sc.GetCloudCredentials("aws")
		require.NoError(t, err)
		assert.NotNil(t, creds)
		assert.Equal(t, "aws", creds.Provider)
		assert.NotNil(t, creds.AWS)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", creds.AWS.AccessKeyID)
		assert.Equal(t, "us-west-2", creds.AWS.Region)
	})
	
	t.Run("Azure credentials", func(t *testing.T) {
		os.Setenv("DRIFTMGR_AZURE_TENANT_ID", "tenant-123")
		os.Setenv("DRIFTMGR_AZURE_CLIENT_ID", "client-456")
		os.Setenv("DRIFTMGR_AZURE_CLIENT_SECRET", "secret-789")
		os.Setenv("DRIFTMGR_AZURE_SUBSCRIPTION_ID", "sub-000")
		defer func() {
			os.Unsetenv("DRIFTMGR_AZURE_TENANT_ID")
			os.Unsetenv("DRIFTMGR_AZURE_CLIENT_ID")
			os.Unsetenv("DRIFTMGR_AZURE_CLIENT_SECRET")
			os.Unsetenv("DRIFTMGR_AZURE_SUBSCRIPTION_ID")
		}()
		
		sc.loadEnvironmentVariables()
		
		creds, err := sc.GetCloudCredentials("azure")
		require.NoError(t, err)
		assert.NotNil(t, creds)
		assert.Equal(t, "azure", creds.Provider)
		assert.NotNil(t, creds.Azure)
		assert.Equal(t, "tenant-123", creds.Azure.TenantID)
		assert.Equal(t, "sub-000", creds.Azure.SubscriptionID)
	})
	
	t.Run("unsupported provider", func(t *testing.T) {
		creds, err := sc.GetCloudCredentials("unknown")
		assert.Error(t, err)
		assert.Nil(t, creds)
	})
}

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"password", true},
		{"user_password", true},
		{"api_key", true},
		{"secret_token", true},
		{"username", false},
		{"email", false},
		{"config_value", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSensitive(tt.key)
			assert.Equal(t, tt.sensitive, result)
		})
	}
}

func TestDefaultValidators(t *testing.T) {
	sc, err := NewSecureConfig()
	require.NoError(t, err)
	
	t.Run("port validator", func(t *testing.T) {
		validator := sc.validators["port"]
		require.NotNil(t, validator)
		
		assert.NoError(t, validator("8080"))
		assert.Error(t, validator("99999"))
		assert.Error(t, validator(123)) // Not a string
	})
	
	t.Run("database_url validator", func(t *testing.T) {
		validator := sc.validators["database_url"]
		require.NotNil(t, validator)
		
		assert.NoError(t, validator("postgres://localhost/db"))
		assert.Error(t, validator("invalid-url"))
		assert.Error(t, validator(123)) // Not a string
	})
	
	t.Run("api_key validator", func(t *testing.T) {
		validator := sc.validators["api_key"]
		require.NotNil(t, validator)
		
		assert.NoError(t, validator("12345678901234567890123456789012"))
		assert.Error(t, validator("short"))
		assert.Error(t, validator(123)) // Not a string
	})
}

func BenchmarkEncryption(b *testing.B) {
	sc, _ := NewSecureConfig()
	plaintext := "sensitive_data_to_encrypt"
	
	b.Run("Encrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = sc.encrypt(plaintext)
		}
	})
	
	encrypted, _ := sc.encrypt(plaintext)
	
	b.Run("Decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = sc.decrypt(encrypted)
		}
	})
}

func BenchmarkGetString(b *testing.B) {
	sc, _ := NewSecureConfig()
	sc.configFile["test_key"] = "test_value"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.GetString("test_key")
	}
}

func BenchmarkValidation(b *testing.B) {
	sc, _ := NewSecureConfig()
	
	// Set up some values to validate
	sc.configFile["port"] = "8080"
	sc.configFile["database_url"] = "postgres://localhost/db"
	sc.configFile["api_key"] = "12345678901234567890123456789012"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sc.Validate()
	}
}