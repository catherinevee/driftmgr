package state

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestStateFile_Validation(t *testing.T) {
	tests := []struct {
		name      string
		stateFile *models.StateFile
		wantErr   bool
	}{
		{
			name: "valid state file",
			stateFile: &models.StateFile{
				ID:           "550e8400-e29b-41d4-a716-446655440000",
				BackendID:    "550e8400-e29b-41d4-a716-446655440001",
				Name:         "test-state",
				Path:         "/path/to/state.tfstate",
				Version:      1,
				Serial:       1,
				Lineage:      "test-lineage",
				Size:         1024,
				Checksum:     "test-checksum",
				LastModified: time.Now(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid state file - missing ID",
			stateFile: &models.StateFile{
				BackendID:    "550e8400-e29b-41d4-a716-446655440001",
				Name:         "test-state",
				Path:         "/path/to/state.tfstate",
				Version:      1,
				Serial:       1,
				Lineage:      "test-lineage",
				Size:         1024,
				Checksum:     "test-checksum",
				LastModified: time.Now(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid state file - missing name",
			stateFile: &models.StateFile{
				ID:           "550e8400-e29b-41d4-a716-446655440000",
				BackendID:    "550e8400-e29b-41d4-a716-446655440001",
				Path:         "/path/to/state.tfstate",
				Version:      1,
				Serial:       1,
				Lineage:      "test-lineage",
				Size:         1024,
				Checksum:     "test-checksum",
				LastModified: time.Now(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.stateFile.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStateResource_Validation(t *testing.T) {
	tests := []struct {
		name     string
		resource *models.StateResource
		wantErr  bool
	}{
		{
			name: "valid resource",
			resource: &models.StateResource{
				ID:            "550e8400-e29b-41d4-a716-446655440002",
				StateFileID:   "550e8400-e29b-41d4-a716-446655440000",
				Address:       "aws_instance.test",
				Type:          "aws_instance",
				Provider:      "aws",
				Instance:      "i-1234567890abcdef0",
				Attributes:    map[string]interface{}{"instance_type": "t2.micro"},
				Mode:          "managed",
				SchemaVersion: 0,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid resource - missing ID",
			resource: &models.StateResource{
				StateFileID:   "550e8400-e29b-41d4-a716-446655440000",
				Address:       "aws_instance.test",
				Type:          "aws_instance",
				Provider:      "aws",
				Instance:      "i-1234567890abcdef0",
				Attributes:    map[string]interface{}{"instance_type": "t2.micro"},
				Mode:          "managed",
				SchemaVersion: 0,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid resource - missing address",
			resource: &models.StateResource{
				ID:            "550e8400-e29b-41d4-a716-446655440002",
				StateFileID:   "550e8400-e29b-41d4-a716-446655440000",
				Type:          "aws_instance",
				Provider:      "aws",
				Instance:      "i-1234567890abcdef0",
				Attributes:    map[string]interface{}{"instance_type": "t2.micro"},
				Mode:          "managed",
				SchemaVersion: 0,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackend_Validation(t *testing.T) {
	tests := []struct {
		name    string
		backend *models.Backend
		wantErr bool
	}{
		{
			name: "valid backend",
			backend: &models.Backend{
				ID:            "550e8400-e29b-41d4-a716-446655440001",
				EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
				Type:          models.BackendTypeS3,
				Name:          "test-backend",
				Description:   "Test backend",
				Configuration: map[string]interface{}{
					"bucket": "test-bucket",
					"key":    "test-key",
				},
				IsActive:  true,
				IsDefault: false,
				CreatedBy: "test-user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid backend - missing ID",
			backend: &models.Backend{
				EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
				Type:          models.BackendTypeS3,
				Name:          "test-backend",
				Description:   "Test backend",
				Configuration: map[string]interface{}{
					"bucket": "test-bucket",
					"key":    "test-key",
				},
				IsActive:  true,
				IsDefault: false,
				CreatedBy: "test-user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid backend - missing name",
			backend: &models.Backend{
				ID:            "550e8400-e29b-41d4-a716-446655440001",
				EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
				Type:          models.BackendTypeS3,
				Description:   "Test backend",
				Configuration: map[string]interface{}{
					"bucket": "test-bucket",
					"key":    "test-key",
				},
				IsActive:  true,
				IsDefault: false,
				CreatedBy: "test-user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStateOperation_Validation(t *testing.T) {
	tests := []struct {
		name      string
		operation *models.StateOperation
		wantErr   bool
	}{
		{
			name: "valid operation",
			operation: &models.StateOperation{
				ID:            "550e8400-e29b-41d4-a716-446655440004",
				StateFileID:   "550e8400-e29b-41d4-a716-446655440000",
				OperationType: models.StateOperationImport,
				Status:        models.OperationStatusCompleted,
				CreatedBy:     "test-user",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid operation - missing ID",
			operation: &models.StateOperation{
				StateFileID:   "550e8400-e29b-41d4-a716-446655440000",
				OperationType: models.StateOperationImport,
				Status:        models.OperationStatusCompleted,
				CreatedBy:     "test-user",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid operation - missing type",
			operation: &models.StateOperation{
				ID:          "550e8400-e29b-41d4-a716-446655440004",
				StateFileID: "550e8400-e29b-41d4-a716-446655440000",
				Status:      models.OperationStatusCompleted,
				CreatedBy:   "test-user",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStateLock_Validation(t *testing.T) {
	tests := []struct {
		name    string
		lock    *models.StateLock
		wantErr bool
	}{
		{
			name: "valid lock",
			lock: &models.StateLock{
				ID:          "550e8400-e29b-41d4-a716-446655440005",
				StateFileID: "550e8400-e29b-41d4-a716-446655440000",
				LockID:      "test-lock-id",
				Operation:   "plan",
				Who:         "test-user",
				Version:     "1.0.0",
				Created:     time.Now(),
				Path:        "/path/to/state.tfstate",
			},
			wantErr: false,
		},
		{
			name: "invalid lock - missing ID",
			lock: &models.StateLock{
				StateFileID: "550e8400-e29b-41d4-a716-446655440000",
				LockID:      "test-lock-id",
				Operation:   "plan",
				Who:         "test-user",
				Version:     "1.0.0",
				Created:     time.Now(),
				Path:        "/path/to/state.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid lock - missing lock ID",
			lock: &models.StateLock{
				ID:          "550e8400-e29b-41d4-a716-446655440005",
				StateFileID: "550e8400-e29b-41d4-a716-446655440000",
				Operation:   "plan",
				Who:         "test-user",
				Version:     "1.0.0",
				Created:     time.Now(),
				Path:        "/path/to/state.tfstate",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lock.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackendType_String(t *testing.T) {
	tests := []struct {
		name        string
		backendType models.BackendType
		expected    string
	}{
		{
			name:        "S3 backend",
			backendType: models.BackendTypeS3,
			expected:    "s3",
		},
		{
			name:        "Azure backend",
			backendType: models.BackendTypeAzure,
			expected:    "azurerm",
		},
		{
			name:        "GCS backend",
			backendType: models.BackendTypeGCS,
			expected:    "gcs",
		},
		{
			name:        "Local backend",
			backendType: models.BackendTypeLocal,
			expected:    "local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.backendType.String())
		})
	}
}

func TestStateOperationType_String(t *testing.T) {
	tests := []struct {
		name     string
		opType   models.StateOperationType
		expected string
	}{
		{
			name:     "Import operation",
			opType:   models.StateOperationImport,
			expected: "import",
		},
		{
			name:     "Remove operation",
			opType:   models.StateOperationRemove,
			expected: "remove",
		},
		{
			name:     "Move operation",
			opType:   models.StateOperationMove,
			expected: "move",
		},
		{
			name:     "List operation",
			opType:   models.StateOperationList,
			expected: "list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.opType.String())
		})
	}
}

func TestOperationStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   models.OperationStatus
		expected string
	}{
		{
			name:     "Pending status",
			status:   models.OperationStatusPending,
			expected: "pending",
		},
		{
			name:     "Running status",
			status:   models.OperationStatusRunning,
			expected: "running",
		},
		{
			name:     "Completed status",
			status:   models.OperationStatusCompleted,
			expected: "completed",
		},
		{
			name:     "Failed status",
			status:   models.OperationStatusFailed,
			expected: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}
