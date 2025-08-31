package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// CloudEventListener listens for real-time cloud resource events
type CloudEventListener struct {
	provider          string
	eventHandlers     map[EventType][]EventHandler
	resourceBuffer    *ResourceEventBuffer
	cloudTrailClient  *cloudtrail.Client
	eventBridgeClient *eventbridge.Client
	sqsClient         *sqs.Client
	queueURL          string
	running           bool
	stopChan          chan struct{}
	mu                sync.RWMutex
}

// EventType represents the type of cloud event
type EventType string

const (
	EventTypeResourceCreated   EventType = "RESOURCE_CREATED"
	EventTypeResourceModified  EventType = "RESOURCE_MODIFIED"
	EventTypeResourceDeleted   EventType = "RESOURCE_DELETED"
	EventTypeTagsModified      EventType = "TAGS_MODIFIED"
	EventTypePermissionChanged EventType = "PERMISSION_CHANGED"
	EventTypeStateChanged      EventType = "STATE_CHANGED"
)

// CloudEvent represents a cloud resource event
type CloudEvent struct {
	ID            string
	Type          EventType
	Provider      string
	Region        string
	AccountID     string
	ResourceID    string
	ResourceType  string
	ResourceName  string
	EventTime     time.Time
	EventSource   string
	UserIdentity  UserIdentity
	RequestParams map[string]interface{}
	ResponseData  map[string]interface{}
	SourceIP      string
	UserAgent     string
	ErrorCode     string
	ErrorMessage  string
	Tags          map[string]string
}

// UserIdentity represents who performed the action
type UserIdentity struct {
	Type        string
	PrincipalID string
	ARN         string
	AccountID   string
	UserName    string
	InvokedBy   string
	SessionContext map[string]interface{}
}

// EventHandler handles cloud events
type EventHandler interface {
	HandleEvent(event CloudEvent) error
	GetPriority() int
}

// ResourceEventBuffer buffers resource events for batch processing
type ResourceEventBuffer struct {
	events       []CloudEvent
	maxSize      int
	flushTimeout time.Duration
	flushFunc    func([]CloudEvent)
	mu           sync.Mutex
	timer        *time.Timer
}

// CloudTrailProcessor processes CloudTrail events
type CloudTrailProcessor struct {
	client           *cloudtrail.Client
	lookbackMinutes  int
	processedEvents  map[string]bool
	resourcePatterns map[string]ResourcePattern
	mu               sync.RWMutex
}

// ResourcePattern defines patterns for resource detection
type ResourcePattern struct {
	EventName      string
	ResourceType   string
	ExtractorFunc  func(map[string]interface{}) (string, string) // Returns ID and Name
	RequiredParams []string
}

// EventBridgeRule represents an EventBridge rule for resource events
type EventBridgeRule struct {
	Name         string
	Description  string
	EventPattern string
	Target       string
	Enabled      bool
}

// NewCloudEventListener creates a new cloud event listener
func NewCloudEventListener(provider string) (*CloudEventListener, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	listener := &CloudEventListener{
		provider:          provider,
		eventHandlers:     make(map[EventType][]EventHandler),
		cloudTrailClient:  cloudtrail.NewFromConfig(cfg),
		eventBridgeClient: eventbridge.NewFromConfig(cfg),
		sqsClient:         sqs.NewFromConfig(cfg),
		stopChan:          make(chan struct{}),
		resourceBuffer:    NewResourceEventBuffer(100, 30*time.Second),
	}
	
	// Initialize resource patterns
	listener.initializeResourcePatterns()
	
	return listener, nil
}

// NewResourceEventBuffer creates a new event buffer
func NewResourceEventBuffer(maxSize int, flushTimeout time.Duration) *ResourceEventBuffer {
	return &ResourceEventBuffer{
		events:       make([]CloudEvent, 0, maxSize),
		maxSize:      maxSize,
		flushTimeout: flushTimeout,
	}
}

// Start starts listening for cloud events
func (cel *CloudEventListener) Start(ctx context.Context) error {
	cel.mu.Lock()
	if cel.running {
		cel.mu.Unlock()
		return fmt.Errorf("listener already running")
	}
	cel.running = true
	cel.mu.Unlock()
	
	// Set up EventBridge rules
	if err := cel.setupEventBridgeRules(ctx); err != nil {
		return fmt.Errorf("failed to setup EventBridge rules: %w", err)
	}
	
	// Set up SQS queue for events
	if err := cel.setupSQSQueue(ctx); err != nil {
		return fmt.Errorf("failed to setup SQS queue: %w", err)
	}
	
	// Start CloudTrail processor
	go cel.processCloudTrailEvents(ctx)
	
	// Start SQS listener
	go cel.listenToSQS(ctx)
	
	// Start event processor
	go cel.processBufferedEvents(ctx)
	
	return nil
}

// Stop stops the event listener
func (cel *CloudEventListener) Stop() {
	cel.mu.Lock()
	defer cel.mu.Unlock()
	
	if cel.running {
		cel.running = false
		close(cel.stopChan)
	}
}

// setupEventBridgeRules sets up EventBridge rules for resource events
func (cel *CloudEventListener) setupEventBridgeRules(ctx context.Context) error {
	rules := cel.getEventBridgeRules()
	
	for _, rule := range rules {
		// Create or update rule
		_, err := cel.eventBridgeClient.PutRule(ctx, &eventbridge.PutRuleInput{
			Name:         aws.String(rule.Name),
			Description:  aws.String(rule.Description),
			EventPattern: aws.String(rule.EventPattern),
			State:        eventbridge.RuleStateEnabled,
		})
		
		if err != nil {
			return fmt.Errorf("failed to create rule %s: %w", rule.Name, err)
		}
		
		// Add SQS target
		if cel.queueURL != "" {
			_, err = cel.eventBridgeClient.PutTargets(ctx, &eventbridge.PutTargetsInput{
				Rule: aws.String(rule.Name),
				Targets: []eventbridge.Target{
					{
						Arn: aws.String(cel.queueURL),
						Id:  aws.String("1"),
					},
				},
			})
			
			if err != nil {
				return fmt.Errorf("failed to add target for rule %s: %w", rule.Name, err)
			}
		}
	}
	
	return nil
}

// getEventBridgeRules returns EventBridge rules for resource monitoring
func (cel *CloudEventListener) getEventBridgeRules() []EventBridgeRule {
	return []EventBridgeRule{
		{
			Name:        "driftmgr-ec2-events",
			Description: "Capture EC2 resource events",
			EventPattern: `{
				"source": ["aws.ec2"],
				"detail-type": [
					"EC2 Instance State-change Notification",
					"EC2 Instance Launch Successful",
					"EC2 Instance Terminate Successful"
				]
			}`,
		},
		{
			Name:        "driftmgr-rds-events",
			Description: "Capture RDS resource events",
			EventPattern: `{
				"source": ["aws.rds"],
				"detail-type": [
					"RDS DB Instance Event",
					"RDS DB Cluster Event"
				]
			}`,
		},
		{
			Name:        "driftmgr-s3-events",
			Description: "Capture S3 bucket events",
			EventPattern: `{
				"source": ["aws.s3"],
				"detail-type": [
					"Bucket Created",
					"Bucket Deleted"
				]
			}`,
		},
		{
			Name:        "driftmgr-iam-events",
			Description: "Capture IAM resource events",
			EventPattern: `{
				"source": ["aws.iam"],
				"detail-type": [
					"AWS API Call via CloudTrail"
				],
				"detail": {
					"eventName": [
						"CreateRole",
						"DeleteRole",
						"CreatePolicy",
						"DeletePolicy",
						"CreateUser",
						"DeleteUser"
					]
				}
			}`,
		},
		{
			Name:        "driftmgr-tag-events",
			Description: "Capture tagging events",
			EventPattern: `{
				"detail-type": ["AWS API Call via CloudTrail"],
				"detail": {
					"eventName": [
						"TagResource",
						"UntagResource",
						"CreateTags",
						"DeleteTags"
					]
				}
			}`,
		},
	}
}

// setupSQSQueue sets up SQS queue for receiving events
func (cel *CloudEventListener) setupSQSQueue(ctx context.Context) error {
	queueName := "driftmgr-events"
	
	// Create queue if it doesn't exist
	result, err := cel.sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]string{
			"MessageRetentionPeriod": "86400", // 1 day
			"VisibilityTimeout":       "60",    // 1 minute
		},
	})
	
	if err != nil {
		// Queue might already exist
		getResult, getErr := cel.sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
			QueueName: aws.String(queueName),
		})
		if getErr != nil {
			return fmt.Errorf("failed to create or get queue: %w", getErr)
		}
		cel.queueURL = *getResult.QueueUrl
	} else {
		cel.queueURL = *result.QueueUrl
	}
	
	return nil
}

// processCloudTrailEvents processes CloudTrail events
func (cel *CloudEventListener) processCloudTrailEvents(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	processor := &CloudTrailProcessor{
		client:          cel.cloudTrailClient,
		lookbackMinutes: 5,
		processedEvents: make(map[string]bool),
	}
	
	for {
		select {
		case <-ticker.C:
			events, err := processor.GetRecentEvents(ctx)
			if err != nil {
				fmt.Printf("Error getting CloudTrail events: %v\n", err)
				continue
			}
			
			for _, event := range events {
				cel.resourceBuffer.Add(event)
			}
			
		case <-cel.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetRecentEvents gets recent CloudTrail events
func (ctp *CloudTrailProcessor) GetRecentEvents(ctx context.Context) ([]CloudEvent, error) {
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(ctp.lookbackMinutes) * time.Minute)
	
	input := &cloudtrail.LookupEventsInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		LookupAttributes: []types.LookupAttribute{
			{
				AttributeKey:   types.LookupAttributeKeyReadOnly,
				AttributeValue: aws.String("false"), // Only write events
			},
		},
	}
	
	var events []CloudEvent
	
	paginator := cloudtrail.NewLookupEventsPaginator(ctp.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		
		for _, ctEvent := range output.Events {
			// Check if already processed
			eventID := *ctEvent.EventId
			ctp.mu.RLock()
			processed := ctp.processedEvents[eventID]
			ctp.mu.RUnlock()
			
			if processed {
				continue
			}
			
			// Convert to CloudEvent
			cloudEvent := ctp.convertToCloudEvent(ctEvent)
			if cloudEvent != nil && ctp.isResourceEvent(cloudEvent) {
				events = append(events, *cloudEvent)
				
				// Mark as processed
				ctp.mu.Lock()
				ctp.processedEvents[eventID] = true
				ctp.mu.Unlock()
			}
		}
	}
	
	// Clean up old processed events
	ctp.cleanupProcessedEvents()
	
	return events, nil
}

// convertToCloudEvent converts CloudTrail event to CloudEvent
func (ctp *CloudTrailProcessor) convertToCloudEvent(ctEvent types.Event) *CloudEvent {
	if ctEvent.EventName == nil {
		return nil
	}
	
	event := &CloudEvent{
		ID:           *ctEvent.EventId,
		EventTime:    *ctEvent.EventTime,
		EventSource:  *ctEvent.EventSource,
		RequestParams: make(map[string]interface{}),
		ResponseData: make(map[string]interface{}),
	}
	
	// Parse event name to determine type
	eventName := *ctEvent.EventName
	event.Type = ctp.determineEventType(eventName)
	
	// Extract user identity
	if ctEvent.Username != nil {
		event.UserIdentity = UserIdentity{
			UserName: *ctEvent.Username,
		}
	}
	
	// Parse CloudTrail record for more details
	if ctEvent.CloudTrailEvent != nil {
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(*ctEvent.CloudTrailEvent), &record); err == nil {
			// Extract request parameters
			if params, ok := record["requestParameters"].(map[string]interface{}); ok {
				event.RequestParams = params
			}
			
			// Extract response elements
			if response, ok := record["responseElements"].(map[string]interface{}); ok {
				event.ResponseData = response
			}
			
			// Extract resource details
			event.ResourceType, event.ResourceID, event.ResourceName = ctp.extractResourceInfo(eventName, event.RequestParams, event.ResponseData)
			
			// Extract region
			if region, ok := record["awsRegion"].(string); ok {
				event.Region = region
			}
			
			// Extract user identity details
			if identity, ok := record["userIdentity"].(map[string]interface{}); ok {
				if principalId, ok := identity["principalId"].(string); ok {
					event.UserIdentity.PrincipalID = principalId
				}
				if arn, ok := identity["arn"].(string); ok {
					event.UserIdentity.ARN = arn
				}
				if identityType, ok := identity["type"].(string); ok {
					event.UserIdentity.Type = identityType
				}
			}
		}
	}
	
	return event
}

// determineEventType determines the event type from event name
func (ctp *CloudTrailProcessor) determineEventType(eventName string) EventType {
	createEvents := []string{"Create", "Launch", "Allocate", "Register", "Put"}
	modifyEvents := []string{"Update", "Modify", "Change", "Set", "Attach", "Detach"}
	deleteEvents := []string{"Delete", "Terminate", "Remove", "Deregister", "Release"}
	tagEvents := []string{"Tag", "Untag"}
	
	for _, pattern := range createEvents {
		if contains(eventName, pattern) {
			return EventTypeResourceCreated
		}
	}
	
	for _, pattern := range deleteEvents {
		if contains(eventName, pattern) {
			return EventTypeResourceDeleted
		}
	}
	
	for _, pattern := range tagEvents {
		if contains(eventName, pattern) {
			return EventTypeTagsModified
		}
	}
	
	for _, pattern := range modifyEvents {
		if contains(eventName, pattern) {
			return EventTypeResourceModified
		}
	}
	
	return EventTypeStateChanged
}

// extractResourceInfo extracts resource information from event
func (ctp *CloudTrailProcessor) extractResourceInfo(eventName string, params, response map[string]interface{}) (string, string, string) {
	// Resource type mapping based on event name
	resourceTypeMap := map[string]string{
		"RunInstances":       "AWS::EC2::Instance",
		"CreateBucket":       "AWS::S3::Bucket",
		"CreateDBInstance":   "AWS::RDS::DBInstance",
		"CreateFunction":     "AWS::Lambda::Function",
		"CreateRole":         "AWS::IAM::Role",
		"CreatePolicy":       "AWS::IAM::Policy",
		"CreateSecurityGroup": "AWS::EC2::SecurityGroup",
		"CreateVpc":          "AWS::EC2::VPC",
		"CreateSubnet":       "AWS::EC2::Subnet",
		"CreateLoadBalancer": "AWS::ElasticLoadBalancingV2::LoadBalancer",
	}
	
	resourceType := ""
	for event, rType := range resourceTypeMap {
		if eventName == event {
			resourceType = rType
			break
		}
	}
	
	// Extract resource ID and name based on type
	resourceID := ""
	resourceName := ""
	
	// Try common patterns in response
	if response != nil {
		// EC2 instances
		if instances, ok := response["instances"].([]interface{}); ok && len(instances) > 0 {
			if instance, ok := instances[0].(map[string]interface{}); ok {
				if id, ok := instance["instanceId"].(string); ok {
					resourceID = id
				}
			}
		}
		
		// Generic ID fields
		idFields := []string{"resourceId", "id", "arn", "instanceId", "bucketName", "functionName", "roleName", "policyArn"}
		for _, field := range idFields {
			if id, ok := response[field].(string); ok {
				resourceID = id
				break
			}
		}
	}
	
	// Try to get from parameters if not in response
	if resourceID == "" && params != nil {
		nameFields := []string{"name", "bucketName", "functionName", "roleName", "groupName"}
		for _, field := range nameFields {
			if name, ok := params[field].(string); ok {
				resourceName = name
				if resourceID == "" {
					resourceID = name
				}
				break
			}
		}
	}
	
	return resourceType, resourceID, resourceName
}

// isResourceEvent checks if event is a resource event
func (ctp *CloudTrailProcessor) isResourceEvent(event CloudEvent) bool {
	// Filter out read-only and internal events
	excludeEvents := []string{
		"Describe", "List", "Get", "Head",
		"AssumeRole", "Decrypt", "GenerateDataKey",
	}
	
	for _, exclude := range excludeEvents {
		if contains(event.EventSource, exclude) {
			return false
		}
	}
	
	// Must have resource information
	return event.ResourceType != "" && (event.ResourceID != "" || event.ResourceName != "")
}

// cleanupProcessedEvents removes old processed events
func (ctp *CloudTrailProcessor) cleanupProcessedEvents() {
	ctp.mu.Lock()
	defer ctp.mu.Unlock()
	
	// Keep only last 10000 events
	if len(ctp.processedEvents) > 10000 {
		ctp.processedEvents = make(map[string]bool)
	}
}

// listenToSQS listens for events from SQS
func (cel *CloudEventListener) listenToSQS(ctx context.Context) {
	for {
		select {
		case <-cel.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			// Receive messages
			result, err := cel.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(cel.queueURL),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20, // Long polling
			})
			
			if err != nil {
				fmt.Printf("Error receiving SQS messages: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			for _, message := range result.Messages {
				// Parse EventBridge event
				var ebEvent map[string]interface{}
				if err := json.Unmarshal([]byte(*message.Body), &ebEvent); err == nil {
					cloudEvent := cel.parseEventBridgeEvent(ebEvent)
					if cloudEvent != nil {
						cel.resourceBuffer.Add(*cloudEvent)
					}
				}
				
				// Delete message
				cel.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(cel.queueURL),
					ReceiptHandle: message.ReceiptHandle,
				})
			}
		}
	}
}

// parseEventBridgeEvent parses EventBridge event
func (cel *CloudEventListener) parseEventBridgeEvent(ebEvent map[string]interface{}) *CloudEvent {
	event := &CloudEvent{
		Provider:      cel.provider,
		RequestParams: make(map[string]interface{}),
		ResponseData:  make(map[string]interface{}),
	}
	
	// Extract common fields
	if id, ok := ebEvent["id"].(string); ok {
		event.ID = id
	}
	
	if timeStr, ok := ebEvent["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			event.EventTime = t
		}
	}
	
	if source, ok := ebEvent["source"].(string); ok {
		event.EventSource = source
	}
	
	if region, ok := ebEvent["region"].(string); ok {
		event.Region = region
	}
	
	if account, ok := ebEvent["account"].(string); ok {
		event.AccountID = account
	}
	
	// Extract detail
	if detail, ok := ebEvent["detail"].(map[string]interface{}); ok {
		// EC2 instance state change
		if instanceId, ok := detail["instance-id"].(string); ok {
			event.ResourceID = instanceId
			event.ResourceType = "AWS::EC2::Instance"
			
			if state, ok := detail["state"].(string); ok {
				event.ResponseData["state"] = state
			}
		}
		
		// RDS events
		if sourceId, ok := detail["SourceIdentifier"].(string); ok {
			event.ResourceID = sourceId
			
			if sourceType, ok := detail["SourceType"].(string); ok {
				switch sourceType {
				case "db-instance":
					event.ResourceType = "AWS::RDS::DBInstance"
				case "db-cluster":
					event.ResourceType = "AWS::RDS::DBCluster"
				}
			}
		}
	}
	
	// Determine event type
	if detailType, ok := ebEvent["detail-type"].(string); ok {
		if contains(detailType, "Launch") || contains(detailType, "Create") {
			event.Type = EventTypeResourceCreated
		} else if contains(detailType, "Terminate") || contains(detailType, "Delete") {
			event.Type = EventTypeResourceDeleted
		} else {
			event.Type = EventTypeStateChanged
		}
	}
	
	return event
}

// processBufferedEvents processes buffered events
func (cel *CloudEventListener) processBufferedEvents(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			events := cel.resourceBuffer.Flush()
			if len(events) > 0 {
				cel.handleEvents(events)
			}
			
		case <-cel.stopChan:
			// Process remaining events
			events := cel.resourceBuffer.Flush()
			if len(events) > 0 {
				cel.handleEvents(events)
			}
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// handleEvents handles a batch of events
func (cel *CloudEventListener) handleEvents(events []CloudEvent) {
	// Group events by type
	eventsByType := make(map[EventType][]CloudEvent)
	for _, event := range events {
		eventsByType[event.Type] = append(eventsByType[event.Type], event)
	}
	
	// Process each type
	for eventType, typeEvents := range eventsByType {
		handlers := cel.eventHandlers[eventType]
		
		// Sort handlers by priority
		sortHandlersByPriority(handlers)
		
		// Execute handlers
		for _, handler := range handlers {
			for _, event := range typeEvents {
				if err := handler.HandleEvent(event); err != nil {
					fmt.Printf("Error handling event %s: %v\n", event.ID, err)
				}
			}
		}
	}
}

// RegisterHandler registers an event handler
func (cel *CloudEventListener) RegisterHandler(eventType EventType, handler EventHandler) {
	cel.mu.Lock()
	defer cel.mu.Unlock()
	
	cel.eventHandlers[eventType] = append(cel.eventHandlers[eventType], handler)
}

// Add adds an event to the buffer
func (reb *ResourceEventBuffer) Add(event CloudEvent) {
	reb.mu.Lock()
	defer reb.mu.Unlock()
	
	reb.events = append(reb.events, event)
	
	// Flush if buffer is full
	if len(reb.events) >= reb.maxSize {
		reb.flush()
	}
	
	// Reset timer
	if reb.timer != nil {
		reb.timer.Stop()
	}
	reb.timer = time.AfterFunc(reb.flushTimeout, func() {
		reb.mu.Lock()
		defer reb.mu.Unlock()
		reb.flush()
	})
}

// Flush flushes the buffer
func (reb *ResourceEventBuffer) Flush() []CloudEvent {
	reb.mu.Lock()
	defer reb.mu.Unlock()
	
	return reb.flush()
}

// flush internal flush function
func (reb *ResourceEventBuffer) flush() []CloudEvent {
	if len(reb.events) == 0 {
		return nil
	}
	
	events := make([]CloudEvent, len(reb.events))
	copy(events, reb.events)
	
	// Clear buffer
	reb.events = reb.events[:0]
	
	// Stop timer
	if reb.timer != nil {
		reb.timer.Stop()
		reb.timer = nil
	}
	
	// Call flush function if set
	if reb.flushFunc != nil {
		reb.flushFunc(events)
	}
	
	return events
}

// SetFlushFunc sets the flush function
func (reb *ResourceEventBuffer) SetFlushFunc(f func([]CloudEvent)) {
	reb.flushFunc = f
}

// initializeResourcePatterns initializes resource detection patterns
func (cel *CloudEventListener) initializeResourcePatterns() {
	// This would be expanded with more patterns
}

// GetRecentEvents returns recent events
func (cel *CloudEventListener) GetRecentEvents(since time.Time) []CloudEvent {
	cel.mu.RLock()
	defer cel.mu.RUnlock()
	
	events := cel.resourceBuffer.events
	recent := []CloudEvent{}
	
	for _, event := range events {
		if event.EventTime.After(since) {
			recent = append(recent, event)
		}
	}
	
	return recent
}

// ConvertToResource converts a CloudEvent to a Resource
func (cel *CloudEventListener) ConvertToResource(event CloudEvent) models.Resource {
	return models.Resource{
		ID:        event.ResourceID,
		Name:      event.ResourceName,
		Type:      event.ResourceType,
		Provider:  event.Provider,
		Region:    event.Region,
		AccountID: event.AccountID,
		Tags:      event.Tags,
		CreatedAt: event.EventTime,
		Metadata: map[string]string{
			"created_by":     event.UserIdentity.UserName,
			"creation_method": detectCreationMethod(event),
			"source_ip":      event.SourceIP,
			"user_agent":     event.UserAgent,
		},
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sortHandlersByPriority(handlers []EventHandler) {
	// Simple priority sort - would use sort.Slice in production
	for i := 0; i < len(handlers); i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[i].GetPriority() < handlers[j].GetPriority() {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}
}

func detectCreationMethod(event CloudEvent) string {
	// Console detection
	if event.UserAgent != "" && contains(event.UserAgent, "console.amazonaws.com") {
		return "console"
	}
	
	// CLI detection
	if event.UserAgent != "" && contains(event.UserAgent, "aws-cli") {
		return "cli"
	}
	
	// Terraform detection
	if event.UserAgent != "" && contains(event.UserAgent, "terraform") {
		return "terraform"
	}
	
	// CloudFormation detection
	if event.UserIdentity.InvokedBy == "cloudformation.amazonaws.com" {
		return "cloudformation"
	}
	
	// SDK detection
	if event.UserAgent != "" && contains(event.UserAgent, "sdk") {
		return "sdk"
	}
	
	return "unknown"
}