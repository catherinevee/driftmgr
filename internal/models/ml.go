package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// MLModelType represents the type of ML model
type MLModelType string

const (
	MLModelTypeDriftPredictor       MLModelType = "drift_predictor"
	MLModelTypeCostOptimizer        MLModelType = "cost_optimizer"
	MLModelTypeAnomalyDetector      MLModelType = "anomaly_detector"
	MLModelTypeResourceClassifier   MLModelType = "resource_classifier"
	MLModelTypeSecurityAnalyzer     MLModelType = "security_analyzer"
	MLModelTypePerformancePredictor MLModelType = "performance_predictor"
	MLModelTypeCapacityPlanner      MLModelType = "capacity_planner"
	MLModelTypeCustom               MLModelType = "custom"
)

// String returns the string representation of MLModelType
func (mlmt MLModelType) String() string {
	return string(mlmt)
}

// MLModelStatus represents the status of an ML model
type MLModelStatus string

const (
	MLModelStatusDraft    MLModelStatus = "draft"
	MLModelStatusTraining MLModelStatus = "training"
	MLModelStatusTrained  MLModelStatus = "trained"
	MLModelStatusDeployed MLModelStatus = "deployed"
	MLModelStatusFailed   MLModelStatus = "failed"
	MLModelStatusRetired  MLModelStatus = "retired"
)

// String returns the string representation of MLModelStatus
func (mlms MLModelStatus) String() string {
	return string(mlms)
}

// MLModel represents an ML model
type MLModel struct {
	ID                string                 `json:"id" db:"id" validate:"required,uuid"`
	Name              string                 `json:"name" db:"name" validate:"required"`
	Type              MLModelType            `json:"type" db:"type" validate:"required"`
	Version           string                 `json:"version" db:"version" validate:"required"`
	Status            MLModelStatus          `json:"status" db:"status" validate:"required"`
	Description       string                 `json:"description" db:"description"`
	Algorithm         string                 `json:"algorithm" db:"algorithm" validate:"required"`
	Accuracy          float64                `json:"accuracy" db:"accuracy"`
	Precision         float64                `json:"precision" db:"precision"`
	Recall            float64                `json:"recall" db:"recall"`
	F1Score           float64                `json:"f1_score" db:"f1_score"`
	TrainingData      map[string]interface{} `json:"training_data" db:"training_data"`
	ModelData         []byte                 `json:"model_data" db:"model_data"`
	Parameters        map[string]interface{} `json:"parameters" db:"parameters"`
	Features          []string               `json:"features" db:"features"`
	Target            string                 `json:"target" db:"target"`
	TrainingMetrics   TrainingMetrics        `json:"training_metrics" db:"training_metrics"`
	ValidationMetrics ValidationMetrics      `json:"validation_metrics" db:"validation_metrics"`
	DeploymentInfo    *DeploymentInfo        `json:"deployment_info" db:"deployment_info"`
	CreatedBy         string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// TrainingMetrics represents training metrics for an ML model
type TrainingMetrics struct {
	Loss          float64            `json:"loss" db:"loss"`
	Accuracy      float64            `json:"accuracy" db:"accuracy"`
	Precision     float64            `json:"precision" db:"precision"`
	Recall        float64            `json:"recall" db:"recall"`
	F1Score       float64            `json:"f1_score" db:"f1_score"`
	Epochs        int                `json:"epochs" db:"epochs"`
	TrainingTime  time.Duration      `json:"training_time" db:"training_time"`
	DataSize      int                `json:"data_size" db:"data_size"`
	Features      int                `json:"features" db:"features"`
	CustomMetrics map[string]float64 `json:"custom_metrics" db:"custom_metrics"`
}

// ValidationMetrics represents validation metrics for an ML model
type ValidationMetrics struct {
	Loss            float64            `json:"loss" db:"loss"`
	Accuracy        float64            `json:"accuracy" db:"accuracy"`
	Precision       float64            `json:"precision" db:"precision"`
	Recall          float64            `json:"recall" db:"recall"`
	F1Score         float64            `json:"f1_score" db:"f1_score"`
	ConfusionMatrix [][]int            `json:"confusion_matrix" db:"confusion_matrix"`
	ROCScore        float64            `json:"roc_score" db:"roc_score"`
	AUCScore        float64            `json:"auc_score" db:"auc_score"`
	CustomMetrics   map[string]float64 `json:"custom_metrics" db:"custom_metrics"`
}

// DeploymentInfo represents deployment information for an ML model
type DeploymentInfo struct {
	Endpoint        string               `json:"endpoint" db:"endpoint"`
	Version         string               `json:"version" db:"version"`
	Environment     string               `json:"environment" db:"environment"`
	Replicas        int                  `json:"replicas" db:"replicas"`
	Resources       ResourceRequirements `json:"resources" db:"resources"`
	HealthCheck     HealthCheckConfig    `json:"health_check" db:"health_check"`
	ScalingConfig   ScalingConfig        `json:"scaling_config" db:"scaling_config"`
	DeployedAt      time.Time            `json:"deployed_at" db:"deployed_at"`
	LastHealthCheck *time.Time           `json:"last_health_check" db:"last_health_check"`
	Status          DeploymentStatus     `json:"status" db:"status"`
}

// ResourceRequirements represents resource requirements for deployment
type ResourceRequirements struct {
	CPU     string `json:"cpu" db:"cpu"`
	Memory  string `json:"memory" db:"memory"`
	GPU     string `json:"gpu" db:"gpu"`
	Storage string `json:"storage" db:"storage"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Path             string        `json:"path" db:"path"`
	Interval         time.Duration `json:"interval" db:"interval"`
	Timeout          time.Duration `json:"timeout" db:"timeout"`
	Retries          int           `json:"retries" db:"retries"`
	SuccessThreshold int           `json:"success_threshold" db:"success_threshold"`
	FailureThreshold int           `json:"failure_threshold" db:"failure_threshold"`
}

// ScalingConfig represents scaling configuration
type ScalingConfig struct {
	MinReplicas  int `json:"min_replicas" db:"min_replicas"`
	MaxReplicas  int `json:"max_replicas" db:"max_replicas"`
	TargetCPU    int `json:"target_cpu" db:"target_cpu"`
	TargetMemory int `json:"target_memory" db:"target_memory"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusDeploying DeploymentStatus = "deploying"
	DeploymentStatusActive    DeploymentStatus = "active"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusStopped   DeploymentStatus = "stopped"
)

// String returns the string representation of DeploymentStatus
func (ds DeploymentStatus) String() string {
	return string(ds)
}

// MLPrediction represents a prediction made by an ML model
type MLPrediction struct {
	ID             string                 `json:"id" db:"id" validate:"required,uuid"`
	ModelID        string                 `json:"model_id" db:"model_id" validate:"required,uuid"`
	InputData      map[string]interface{} `json:"input_data" db:"input_data"`
	Prediction     interface{}            `json:"prediction" db:"prediction"`
	Confidence     float64                `json:"confidence" db:"confidence"`
	Probability    map[string]float64     `json:"probability" db:"probability"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	ProcessingTime time.Duration          `json:"processing_time" db:"processing_time"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// MLTrainingJob represents an ML training job
type MLTrainingJob struct {
	ID             string                 `json:"id" db:"id" validate:"required,uuid"`
	ModelID        string                 `json:"model_id" db:"model_id" validate:"required,uuid"`
	Status         TrainingJobStatus      `json:"status" db:"status" validate:"required"`
	TrainingData   TrainingDataConfig     `json:"training_data" db:"training_data"`
	ValidationData ValidationDataConfig   `json:"validation_data" db:"validation_data"`
	Parameters     map[string]interface{} `json:"parameters" db:"parameters"`
	Progress       TrainingProgress       `json:"progress" db:"progress"`
	StartedAt      *time.Time             `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at" db:"completed_at"`
	Error          *string                `json:"error,omitempty" db:"error"`
	CreatedBy      string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// TrainingJobStatus represents the status of a training job
type TrainingJobStatus string

const (
	TrainingJobStatusPending   TrainingJobStatus = "pending"
	TrainingJobStatusRunning   TrainingJobStatus = "running"
	TrainingJobStatusCompleted TrainingJobStatus = "completed"
	TrainingJobStatusFailed    TrainingJobStatus = "failed"
	TrainingJobStatusCancelled TrainingJobStatus = "cancelled"
)

// String returns the string representation of TrainingJobStatus
func (tjs TrainingJobStatus) String() string {
	return string(tjs)
}

// TrainingDataConfig represents training data configuration
type TrainingDataConfig struct {
	Source          string                 `json:"source" db:"source" validate:"required"`
	Query           string                 `json:"query" db:"query"`
	Filters         []AnalyticsFilter      `json:"filters" db:"filters"`
	TimeRange       TimeRange              `json:"time_range" db:"time_range"`
	Features        []string               `json:"features" db:"features"`
	Target          string                 `json:"target" db:"target"`
	Preprocessing   map[string]interface{} `json:"preprocessing" db:"preprocessing"`
	ValidationSplit float64                `json:"validation_split" db:"validation_split"`
}

// ValidationDataConfig represents validation data configuration
type ValidationDataConfig struct {
	Source        string                 `json:"source" db:"source"`
	Query         string                 `json:"query" db:"query"`
	Filters       []AnalyticsFilter      `json:"filters" db:"filters"`
	TimeRange     TimeRange              `json:"time_range" db:"time_range"`
	Preprocessing map[string]interface{} `json:"preprocessing" db:"preprocessing"`
}

// TrainingProgress represents the progress of a training job
type TrainingProgress struct {
	CurrentEpoch    int     `json:"current_epoch" db:"current_epoch"`
	TotalEpochs     int     `json:"total_epochs" db:"total_epochs"`
	CurrentLoss     float64 `json:"current_loss" db:"current_loss"`
	CurrentAccuracy float64 `json:"current_accuracy" db:"current_accuracy"`
	Percentage      float64 `json:"percentage" db:"percentage"`
	EstimatedTime   string  `json:"estimated_time" db:"estimated_time"`
}

// MLInference represents an ML inference request
type MLInference struct {
	ID          string                 `json:"id" db:"id" validate:"required,uuid"`
	ModelID     string                 `json:"model_id" db:"model_id" validate:"required,uuid"`
	InputData   map[string]interface{} `json:"input_data" db:"input_data" validate:"required"`
	BatchSize   int                    `json:"batch_size" db:"batch_size"`
	Async       bool                   `json:"async" db:"async"`
	CallbackURL string                 `json:"callback_url" db:"callback_url"`
	Status      InferenceStatus        `json:"status" db:"status"`
	Result      *MLPrediction          `json:"result" db:"result"`
	Error       *string                `json:"error,omitempty" db:"error"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	CompletedAt *time.Time             `json:"completed_at" db:"completed_at"`
}

// InferenceStatus represents the status of an inference
type InferenceStatus string

const (
	InferenceStatusPending   InferenceStatus = "pending"
	InferenceStatusRunning   InferenceStatus = "running"
	InferenceStatusCompleted InferenceStatus = "completed"
	InferenceStatusFailed    InferenceStatus = "failed"
)

// String returns the string representation of InferenceStatus
func (is InferenceStatus) String() string {
	return string(is)
}

// Request/Response Models

// MLModelCreateRequest represents a request to create an ML model
type MLModelCreateRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Type        MLModelType            `json:"type" validate:"required"`
	Description string                 `json:"description"`
	Algorithm   string                 `json:"algorithm" validate:"required"`
	Parameters  map[string]interface{} `json:"parameters"`
	Features    []string               `json:"features"`
	Target      string                 `json:"target"`
}

// MLModelListRequest represents a request to list ML models
type MLModelListRequest struct {
	Type      *MLModelType   `json:"type,omitempty"`
	Status    *MLModelStatus `json:"status,omitempty"`
	CreatedBy *string        `json:"created_by,omitempty"`
	Limit     int            `json:"limit" validate:"min=1,max=1000"`
	Offset    int            `json:"offset" validate:"min=0"`
	SortBy    string         `json:"sort_by" validate:"omitempty,oneof=name created_at updated_at accuracy"`
	SortOrder string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// MLModelListResponse represents the response for listing ML models
type MLModelListResponse struct {
	Models []MLModel `json:"models"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

// MLModelTrainRequest represents a request to train an ML model
type MLModelTrainRequest struct {
	TrainingData   TrainingDataConfig     `json:"training_data" validate:"required"`
	ValidationData *ValidationDataConfig  `json:"validation_data"`
	Parameters     map[string]interface{} `json:"parameters"`
	Async          bool                   `json:"async"`
}

// MLModelPredictRequest represents a request to make a prediction
type MLModelPredictRequest struct {
	InputData   map[string]interface{} `json:"input_data" validate:"required"`
	BatchSize   int                    `json:"batch_size"`
	Async       bool                   `json:"async"`
	CallbackURL string                 `json:"callback_url"`
}

// MLModelPredictResponse represents the response for a prediction
type MLModelPredictResponse struct {
	Prediction     interface{}            `json:"prediction"`
	Confidence     float64                `json:"confidence"`
	Probability    map[string]float64     `json:"probability"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// Validation methods

// Validate validates the MLModel struct
func (mlm *MLModel) Validate() error {
	validate := validator.New()
	return validate.Struct(mlm)
}

// Validate validates the MLPrediction struct
func (mlp *MLPrediction) Validate() error {
	validate := validator.New()
	return validate.Struct(mlp)
}

// Validate validates the MLTrainingJob struct
func (mltj *MLTrainingJob) Validate() error {
	validate := validator.New()
	return validate.Struct(mltj)
}

// Validate validates the MLInference struct
func (mli *MLInference) Validate() error {
	validate := validator.New()
	return validate.Struct(mli)
}

// Validate validates the MLModelCreateRequest struct
func (mlmcr *MLModelCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(mlmcr)
}

// Validate validates the MLModelListRequest struct
func (mlmlr *MLModelListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(mlmlr)
}

// Validate validates the MLModelTrainRequest struct
func (mlmtr *MLModelTrainRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(mlmtr)
}

// Validate validates the MLModelPredictRequest struct
func (mlmpr *MLModelPredictRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(mlmpr)
}

// Helper methods

// IsDeployed returns true if the model is deployed
func (mlm *MLModel) IsDeployed() bool {
	return mlm.Status == MLModelStatusDeployed && mlm.DeploymentInfo != nil
}

// IsTraining returns true if the model is training
func (mlm *MLModel) IsTraining() bool {
	return mlm.Status == MLModelStatusTraining
}

// IsTrained returns true if the model is trained
func (mlm *MLModel) IsTrained() bool {
	return mlm.Status == MLModelStatusTrained || mlm.Status == MLModelStatusDeployed
}

// GetAccuracy returns the model accuracy
func (mlm *MLModel) GetAccuracy() float64 {
	return mlm.Accuracy
}

// GetF1Score returns the model F1 score
func (mlm *MLModel) GetF1Score() float64 {
	return mlm.F1Score
}

// IsCompleted returns true if the training job is completed
func (mltj *MLTrainingJob) IsCompleted() bool {
	return mltj.Status == TrainingJobStatusCompleted
}

// IsFailed returns true if the training job failed
func (mltj *MLTrainingJob) IsFailed() bool {
	return mltj.Status == TrainingJobStatusFailed
}

// IsRunning returns true if the training job is running
func (mltj *MLTrainingJob) IsRunning() bool {
	return mltj.Status == TrainingJobStatusRunning
}

// GetProgress returns the training progress
func (mltj *MLTrainingJob) GetProgress() TrainingProgress {
	return mltj.Progress
}

// IsCompleted returns true if the inference is completed
func (mli *MLInference) IsCompleted() bool {
	return mli.Status == InferenceStatusCompleted
}

// IsFailed returns true if the inference failed
func (mli *MLInference) IsFailed() bool {
	return mli.Status == InferenceStatusFailed
}

// IsRunning returns true if the inference is running
func (mli *MLInference) IsRunning() bool {
	return mli.Status == InferenceStatusRunning
}

// GetResult returns the inference result
func (mli *MLInference) GetResult() *MLPrediction {
	return mli.Result
}

// HasError returns true if the inference has an error
func (mli *MLInference) HasError() bool {
	return mli.Error != nil
}

// GetError returns the error message
func (mli *MLInference) GetError() string {
	if mli.Error == nil {
		return ""
	}
	return *mli.Error
}

// SetError sets the error message
func (mli *MLInference) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		mli.Error = &errStr
		mli.Status = InferenceStatusFailed
	}
}
