package history

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/graph"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

type ChangeType string

const (
	ChangeTypeCreated   ChangeType = "created"
	ChangeTypeModified  ChangeType = "modified"
	ChangeTypeDeleted   ChangeType = "deleted"
	ChangeTypeDrifted   ChangeType = "drifted"
	ChangeTypeImported  ChangeType = "imported"
	ChangeTypeRecreated ChangeType = "recreated"
)

type ResourceChange struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	ChangeType   ChangeType             `json:"change_type"`
	Timestamp    time.Time              `json:"timestamp"`
	OldState     map[string]interface{} `json:"old_state,omitempty"`
	NewState     map[string]interface{} `json:"new_state,omitempty"`
	Diff         map[string]interface{} `json:"diff,omitempty"`
	Actor        string                 `json:"actor,omitempty"`
	Source       string                 `json:"source"`
	StateFile    string                 `json:"state_file,omitempty"`
	Confidence   float64                `json:"confidence"`
}

type StateSnapshot struct {
	ID           string                   `json:"id"`
	Timestamp    time.Time                `json:"timestamp"`
	StateFile    string                   `json:"state_file"`
	Resources    []models.Resource        `json:"resources"`
	Metadata     map[string]interface{}   `json:"metadata"`
	Hash         string                   `json:"hash"`
	Previous     string                   `json:"previous,omitempty"`
}

type ResourceLifecycle struct {
	ResourceID    string           `json:"resource_id"`
	ResourceType  string           `json:"resource_type"`
	Provider      string           `json:"provider"`
	FirstSeen     time.Time        `json:"first_seen"`
	LastSeen      time.Time        `json:"last_seen"`
	Changes       []ResourceChange `json:"changes"`
	StateHistory  []StateSnapshot  `json:"state_history"`
	TotalChanges  int              `json:"total_changes"`
	DriftCount    int              `json:"drift_count"`
	RecreateCount int              `json:"recreate_count"`
	Stability     float64          `json:"stability"`
}

type StateHistoryTracker struct {
	mu              sync.RWMutex
	snapshots       map[string]*StateSnapshot
	changes         []ResourceChange
	lifecycles      map[string]*ResourceLifecycle
	stateLoader     *state.Loader
	graph           *graph.ResourceGraph
	storageDir      string
	retentionDays   int
	maxSnapshots    int
}

func NewStateHistoryTracker(storageDir string) *StateHistoryTracker {
	return &StateHistoryTracker{
		snapshots:     make(map[string]*StateSnapshot),
		changes:       []ResourceChange{},
		lifecycles:    make(map[string]*ResourceLifecycle),
		stateLoader:   state.NewLoader(),
		graph:         graph.NewResourceGraph(),
		storageDir:    storageDir,
		retentionDays: 90,
		maxSnapshots:  1000,
	}
}

func (sht *StateHistoryTracker) LoadHistoricalStates(ctx context.Context) error {
	// Load from Git history
	if err := sht.loadGitHistory(ctx); err != nil {
		return fmt.Errorf("failed to load git history: %w", err)
	}

	// Load from backup directories
	if err := sht.loadBackupHistory(ctx); err != nil {
		return fmt.Errorf("failed to load backup history: %w", err)
	}

	// Load from CI/CD artifacts
	if err := sht.loadCICDHistory(ctx); err != nil {
		return fmt.Errorf("failed to load CI/CD history: %w", err)
	}

	// Load from S3 versioning
	if err := sht.loadS3History(ctx); err != nil {
		return fmt.Errorf("failed to load S3 history: %w", err)
	}

	// Reconstruct missing history
	if err := sht.reconstructMissingHistory(ctx); err != nil {
		return fmt.Errorf("failed to reconstruct history: %w", err)
	}

	return nil
}

func (sht *StateHistoryTracker) loadGitHistory(ctx context.Context) error {
	// Find all terraform.tfstate files in git history
	gitRoot, err := sht.findGitRoot()
	if err != nil {
		return err
	}

	// Get list of commits that modified state files
	commits, err := sht.getStateCommits(gitRoot)
	if err != nil {
		return err
	}

	for _, commit := range commits {
		// Extract state file from specific commit
		stateData, err := sht.extractStateFromCommit(gitRoot, commit)
		if err != nil {
			continue
		}

		snapshot := sht.createSnapshot(stateData, commit.Timestamp, commit.Hash)
		sht.addSnapshot(snapshot)
	}

	return nil
}

func (sht *StateHistoryTracker) loadBackupHistory(ctx context.Context) error {
	backupDirs := []string{
		".terraform.backup",
		"terraform.tfstate.backup",
		"state-backups",
		".terraform/backup",
		"backups",
	}

	for _, dir := range backupDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		files, err := filepath.Glob(filepath.Join(dir, "*.tfstate*"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}

			info, _ := os.Stat(file)
			snapshot := sht.createSnapshot(data, info.ModTime(), filepath.Base(file))
			sht.addSnapshot(snapshot)
		}
	}

	return nil
}

func (sht *StateHistoryTracker) loadCICDHistory(ctx context.Context) error {
	ciDirs := []string{
		".github/artifacts",
		".gitlab/artifacts",
		"jenkins/artifacts",
		"circleci/artifacts",
		"azure-pipelines/artifacts",
	}

	for _, dir := range ciDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if strings.Contains(path, "tfstate") {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					return nil
				}

				snapshot := sht.createSnapshot(data, info.ModTime(), filepath.Base(path))
				sht.addSnapshot(snapshot)
			}

			return nil
		})

		if err != nil {
			continue
		}
	}

	return nil
}

func (sht *StateHistoryTracker) loadS3History(ctx context.Context) error {
	// This would connect to S3 and load versioned state files
	// Implementation depends on AWS SDK and bucket configuration
	return nil
}

func (sht *StateHistoryTracker) reconstructMissingHistory(ctx context.Context) error {
	sht.mu.Lock()
	defer sht.mu.Unlock()

	// Sort snapshots by timestamp
	snapshots := sht.getSortedSnapshots()
	
	for i := 0; i < len(snapshots)-1; i++ {
		current := snapshots[i]
		next := snapshots[i+1]
		
		// Check if there's a gap
		gap := next.Timestamp.Sub(current.Timestamp)
		if gap > 24*time.Hour {
			// Attempt to reconstruct intermediate states
			sht.interpolateStates(current, next)
		}
	}

	return nil
}

func (sht *StateHistoryTracker) interpolateStates(prev, next *StateSnapshot) {
	// Analyze changes between snapshots
	changes := sht.detectChanges(prev, next)
	
	// Estimate intermediate states based on change patterns
	for _, change := range changes {
		// Create synthetic intermediate state
		if change.ChangeType == ChangeTypeModified {
			// Gradual changes likely happened over time
			intermediateTime := prev.Timestamp.Add(next.Timestamp.Sub(prev.Timestamp) / 2)
			sht.addChange(ResourceChange{
				ResourceID:   change.ResourceID,
				ResourceType: change.ResourceType,
				Provider:     change.Provider,
				ChangeType:   ChangeTypeModified,
				Timestamp:    intermediateTime,
				Source:       "reconstructed",
				Confidence:   0.6,
			})
		}
	}
}

func (sht *StateHistoryTracker) TrackChange(change ResourceChange) {
	sht.mu.Lock()
	defer sht.mu.Unlock()

	change.Timestamp = time.Now()
	sht.changes = append(sht.changes, change)

	// Update lifecycle
	lifecycleID := fmt.Sprintf("%s/%s", change.Provider, change.ResourceID)
	lifecycle, exists := sht.lifecycles[lifecycleID]
	if !exists {
		lifecycle = &ResourceLifecycle{
			ResourceID:   change.ResourceID,
			ResourceType: change.ResourceType,
			Provider:     change.Provider,
			FirstSeen:    change.Timestamp,
			LastSeen:     change.Timestamp,
			Changes:      []ResourceChange{},
		}
		sht.lifecycles[lifecycleID] = lifecycle
	}

	lifecycle.Changes = append(lifecycle.Changes, change)
	lifecycle.LastSeen = change.Timestamp
	lifecycle.TotalChanges++

	if change.ChangeType == ChangeTypeDrifted {
		lifecycle.DriftCount++
	}
	if change.ChangeType == ChangeTypeRecreated {
		lifecycle.RecreateCount++
	}

	// Calculate stability score
	lifecycle.Stability = sht.calculateStability(lifecycle)

	// Persist to disk
	sht.persistChange(change)
}

func (sht *StateHistoryTracker) calculateStability(lifecycle *ResourceLifecycle) float64 {
	if lifecycle.TotalChanges == 0 {
		return 1.0
	}

	age := time.Since(lifecycle.FirstSeen).Hours() / 24 // days
	if age == 0 {
		age = 1
	}

	// Calculate change frequency
	changeFrequency := float64(lifecycle.TotalChanges) / age
	
	// Penalize for drifts and recreates
	driftPenalty := float64(lifecycle.DriftCount) * 0.1
	recreatePenalty := float64(lifecycle.RecreateCount) * 0.2
	
	// Calculate stability (0-1 scale)
	stability := 1.0 - (changeFrequency * 0.01) - driftPenalty - recreatePenalty
	
	if stability < 0 {
		stability = 0
	}
	if stability > 1 {
		stability = 1
	}
	
	return stability
}

func (sht *StateHistoryTracker) GetResourceLifecycle(resourceID, provider string) (*ResourceLifecycle, error) {
	sht.mu.RLock()
	defer sht.mu.RUnlock()

	lifecycleID := fmt.Sprintf("%s/%s", provider, resourceID)
	lifecycle, exists := sht.lifecycles[lifecycleID]
	if !exists {
		return nil, fmt.Errorf("lifecycle not found for resource %s", lifecycleID)
	}

	return lifecycle, nil
}

func (sht *StateHistoryTracker) GetChangeHistory(startTime, endTime time.Time) []ResourceChange {
	sht.mu.RLock()
	defer sht.mu.RUnlock()

	filtered := []ResourceChange{}
	for _, change := range sht.changes {
		if change.Timestamp.After(startTime) && change.Timestamp.Before(endTime) {
			filtered = append(filtered, change)
		}
	}

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

func (sht *StateHistoryTracker) GetDriftTrends() map[string]interface{} {
	sht.mu.RLock()
	defer sht.mu.RUnlock()

	// Analyze drift patterns over time
	dailyDrifts := make(map[string]int)
	hourlyDrifts := make(map[int]int)
	providerDrifts := make(map[string]int)
	typeDrifts := make(map[string]int)

	for _, change := range sht.changes {
		if change.ChangeType == ChangeTypeDrifted {
			// Daily aggregation
			day := change.Timestamp.Format("2006-01-02")
			dailyDrifts[day]++

			// Hourly pattern
			hour := change.Timestamp.Hour()
			hourlyDrifts[hour]++

			// Provider breakdown
			providerDrifts[change.Provider]++

			// Resource type breakdown
			typeDrifts[change.ResourceType]++
		}
	}

	// Find peak drift times
	peakHour := 0
	maxHourlyDrifts := 0
	for hour, count := range hourlyDrifts {
		if count > maxHourlyDrifts {
			peakHour = hour
			maxHourlyDrifts = count
		}
	}

	// Calculate drift rate
	totalDays := len(dailyDrifts)
	totalDrifts := 0
	for _, count := range dailyDrifts {
		totalDrifts += count
	}
	
	avgDriftRate := 0.0
	if totalDays > 0 {
		avgDriftRate = float64(totalDrifts) / float64(totalDays)
	}

	return map[string]interface{}{
		"daily_drifts":     dailyDrifts,
		"hourly_pattern":   hourlyDrifts,
		"peak_hour":        peakHour,
		"provider_drifts":  providerDrifts,
		"type_drifts":      typeDrifts,
		"avg_drift_rate":   avgDriftRate,
		"total_drifts":     totalDrifts,
	}
}

func (sht *StateHistoryTracker) PredictNextDrift(resourceID, provider string) (*DriftPrediction, error) {
	lifecycle, err := sht.GetResourceLifecycle(resourceID, provider)
	if err != nil {
		return nil, err
	}

	// Analyze historical drift patterns
	driftIntervals := []time.Duration{}
	lastDrift := time.Time{}
	
	for _, change := range lifecycle.Changes {
		if change.ChangeType == ChangeTypeDrifted {
			if !lastDrift.IsZero() {
				interval := change.Timestamp.Sub(lastDrift)
				driftIntervals = append(driftIntervals, interval)
			}
			lastDrift = change.Timestamp
		}
	}

	if len(driftIntervals) == 0 {
		return nil, fmt.Errorf("insufficient drift history for prediction")
	}

	// Calculate average drift interval
	var totalInterval time.Duration
	for _, interval := range driftIntervals {
		totalInterval += interval
	}
	avgInterval := totalInterval / time.Duration(len(driftIntervals))

	// Predict next drift
	predictedTime := lastDrift.Add(avgInterval)
	confidence := sht.calculatePredictionConfidence(driftIntervals)

	return &DriftPrediction{
		ResourceID:    resourceID,
		Provider:      provider,
		PredictedTime: predictedTime,
		Confidence:    confidence,
		BasedOn:       len(driftIntervals),
	}, nil
}

type DriftPrediction struct {
	ResourceID    string    `json:"resource_id"`
	Provider      string    `json:"provider"`
	PredictedTime time.Time `json:"predicted_time"`
	Confidence    float64   `json:"confidence"`
	BasedOn       int       `json:"based_on"`
}

func (sht *StateHistoryTracker) calculatePredictionConfidence(intervals []time.Duration) float64 {
	if len(intervals) < 2 {
		return 0.1
	}

	// Calculate standard deviation
	var sum, mean float64
	for _, interval := range intervals {
		sum += interval.Seconds()
	}
	mean = sum / float64(len(intervals))

	var variance float64
	for _, interval := range intervals {
		diff := interval.Seconds() - mean
		variance += diff * diff
	}
	variance /= float64(len(intervals))
	stdDev := variance // simplified

	// Lower standard deviation = higher confidence
	confidence := 1.0 / (1.0 + stdDev/mean)
	
	// Adjust based on sample size
	sampleBonus := float64(len(intervals)) / 100.0
	if sampleBonus > 0.2 {
		sampleBonus = 0.2
	}
	confidence += sampleBonus

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (sht *StateHistoryTracker) GetUnstableResources(threshold float64) []*ResourceLifecycle {
	sht.mu.RLock()
	defer sht.mu.RUnlock()

	unstable := []*ResourceLifecycle{}
	
	for _, lifecycle := range sht.lifecycles {
		if lifecycle.Stability < threshold {
			unstable = append(unstable, lifecycle)
		}
	}

	// Sort by stability (least stable first)
	sort.Slice(unstable, func(i, j int) bool {
		return unstable[i].Stability < unstable[j].Stability
	})

	return unstable
}

func (sht *StateHistoryTracker) GenerateTimelineReport(resourceID, provider string) (*TimelineReport, error) {
	lifecycle, err := sht.GetResourceLifecycle(resourceID, provider)
	if err != nil {
		return nil, err
	}

	report := &TimelineReport{
		ResourceID:   resourceID,
		Provider:     provider,
		FirstSeen:    lifecycle.FirstSeen,
		LastSeen:     lifecycle.LastSeen,
		TotalChanges: lifecycle.TotalChanges,
		Events:       []TimelineEvent{},
	}

	for _, change := range lifecycle.Changes {
		event := TimelineEvent{
			Timestamp:   change.Timestamp,
			Type:        string(change.ChangeType),
			Description: sht.describeChange(change),
			Impact:      sht.assessImpact(change),
		}
		report.Events = append(report.Events, event)
	}

	return report, nil
}

type TimelineReport struct {
	ResourceID   string          `json:"resource_id"`
	Provider     string          `json:"provider"`
	FirstSeen    time.Time       `json:"first_seen"`
	LastSeen     time.Time       `json:"last_seen"`
	TotalChanges int             `json:"total_changes"`
	Events       []TimelineEvent `json:"events"`
}

type TimelineEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
}

func (sht *StateHistoryTracker) describeChange(change ResourceChange) string {
	switch change.ChangeType {
	case ChangeTypeCreated:
		return fmt.Sprintf("Resource %s created", change.ResourceID)
	case ChangeTypeModified:
		return fmt.Sprintf("Resource %s modified", change.ResourceID)
	case ChangeTypeDeleted:
		return fmt.Sprintf("Resource %s deleted", change.ResourceID)
	case ChangeTypeDrifted:
		return fmt.Sprintf("Resource %s drifted from desired state", change.ResourceID)
	case ChangeTypeImported:
		return fmt.Sprintf("Resource %s imported into Terraform", change.ResourceID)
	case ChangeTypeRecreated:
		return fmt.Sprintf("Resource %s recreated", change.ResourceID)
	default:
		return fmt.Sprintf("Resource %s changed", change.ResourceID)
	}
}

func (sht *StateHistoryTracker) assessImpact(change ResourceChange) string {
	switch change.ChangeType {
	case ChangeTypeDeleted, ChangeTypeRecreated:
		return "high"
	case ChangeTypeDrifted, ChangeTypeModified:
		return "medium"
	case ChangeTypeCreated, ChangeTypeImported:
		return "low"
	default:
		return "unknown"
	}
}

// Helper methods

func (sht *StateHistoryTracker) findGitRoot() (string, error) {
	// Walk up directory tree to find .git directory
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

type GitCommit struct {
	Hash      string
	Timestamp time.Time
	Message   string
}

func (sht *StateHistoryTracker) getStateCommits(gitRoot string) ([]GitCommit, error) {
	// This would use git commands to get commits that modified state files
	// Simplified implementation
	return []GitCommit{}, nil
}

func (sht *StateHistoryTracker) extractStateFromCommit(gitRoot string, commit GitCommit) ([]byte, error) {
	// This would use git show to extract state file content from specific commit
	// Simplified implementation
	return []byte{}, nil
}

func (sht *StateHistoryTracker) createSnapshot(data []byte, timestamp time.Time, source string) *StateSnapshot {
	// Parse state data and create snapshot
	var stateData map[string]interface{}
	json.Unmarshal(data, &stateData)

	snapshot := &StateSnapshot{
		ID:        fmt.Sprintf("%s-%d", source, timestamp.Unix()),
		Timestamp: timestamp,
		StateFile: source,
		Resources: []models.Resource{},
		Metadata:  stateData,
		Hash:      sht.hashState(data),
	}

	return snapshot
}

func (sht *StateHistoryTracker) hashState(data []byte) string {
	// Calculate hash of state data
	return fmt.Sprintf("%x", data[:32])
}

func (sht *StateHistoryTracker) addSnapshot(snapshot *StateSnapshot) {
	sht.mu.Lock()
	defer sht.mu.Unlock()

	sht.snapshots[snapshot.ID] = snapshot

	// Enforce retention policy
	if len(sht.snapshots) > sht.maxSnapshots {
		sht.pruneOldSnapshots()
	}
}

func (sht *StateHistoryTracker) pruneOldSnapshots() {
	cutoff := time.Now().AddDate(0, 0, -sht.retentionDays)
	
	for id, snapshot := range sht.snapshots {
		if snapshot.Timestamp.Before(cutoff) {
			delete(sht.snapshots, id)
		}
	}
}

func (sht *StateHistoryTracker) getSortedSnapshots() []*StateSnapshot {
	snapshots := []*StateSnapshot{}
	for _, snapshot := range sht.snapshots {
		snapshots = append(snapshots, snapshot)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})

	return snapshots
}

func (sht *StateHistoryTracker) detectChanges(prev, next *StateSnapshot) []ResourceChange {
	changes := []ResourceChange{}
	
	// Compare resource lists between snapshots
	prevResources := make(map[string]models.Resource)
	for _, res := range prev.Resources {
		prevResources[res.ID] = res
	}

	nextResources := make(map[string]models.Resource)
	for _, res := range next.Resources {
		nextResources[res.ID] = res
	}

	// Detect created resources
	for id, res := range nextResources {
		if _, exists := prevResources[id]; !exists {
			changes = append(changes, ResourceChange{
				ResourceID:   id,
				ResourceType: res.Type,
				Provider:     res.Provider,
				ChangeType:   ChangeTypeCreated,
				Timestamp:    next.Timestamp,
				NewState:     res.Properties,
				Source:       "snapshot_diff",
				Confidence:   0.9,
			})
		}
	}

	// Detect deleted resources
	for id, res := range prevResources {
		if _, exists := nextResources[id]; !exists {
			changes = append(changes, ResourceChange{
				ResourceID:   id,
				ResourceType: res.Type,
				Provider:     res.Provider,
				ChangeType:   ChangeTypeDeleted,
				Timestamp:    next.Timestamp,
				OldState:     res.Properties,
				Source:       "snapshot_diff",
				Confidence:   0.9,
			})
		}
	}

	// Detect modified resources
	for id, nextRes := range nextResources {
		if prevRes, exists := prevResources[id]; exists {
			if !sht.resourcesEqual(prevRes, nextRes) {
				changes = append(changes, ResourceChange{
					ResourceID:   id,
					ResourceType: nextRes.Type,
					Provider:     nextRes.Provider,
					ChangeType:   ChangeTypeModified,
					Timestamp:    next.Timestamp,
					OldState:     prevRes.Properties,
					NewState:     nextRes.Properties,
					Source:       "snapshot_diff",
					Confidence:   0.9,
				})
			}
		}
	}

	return changes
}

func (sht *StateHistoryTracker) resourcesEqual(a, b models.Resource) bool {
	// Compare resource properties
	aJSON, _ := json.Marshal(a.Properties)
	bJSON, _ := json.Marshal(b.Properties)
	return string(aJSON) == string(bJSON)
}

func (sht *StateHistoryTracker) persistChange(change ResourceChange) {
	// Save change to disk
	if sht.storageDir == "" {
		return
	}

	filename := filepath.Join(sht.storageDir, fmt.Sprintf("change_%d.json", time.Now().Unix()))
	data, _ := json.MarshalIndent(change, "", "  ")
	ioutil.WriteFile(filename, data, 0644)
}

func (sht *StateHistoryTracker) LoadPersistedChanges() error {
	if sht.storageDir == "" {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(sht.storageDir, "change_*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		var change ResourceChange
		if err := json.Unmarshal(data, &change); err != nil {
			continue
		}

		sht.changes = append(sht.changes, change)
	}

	// Rebuild lifecycles from loaded changes
	for _, change := range sht.changes {
		lifecycleID := fmt.Sprintf("%s/%s", change.Provider, change.ResourceID)
		if _, exists := sht.lifecycles[lifecycleID]; !exists {
			sht.lifecycles[lifecycleID] = &ResourceLifecycle{
				ResourceID:   change.ResourceID,
				ResourceType: change.ResourceType,
				Provider:     change.Provider,
				FirstSeen:    change.Timestamp,
				Changes:      []ResourceChange{},
			}
		}
		lifecycle := sht.lifecycles[lifecycleID]
		lifecycle.Changes = append(lifecycle.Changes, change)
		lifecycle.TotalChanges++
		if change.Timestamp.After(lifecycle.LastSeen) {
			lifecycle.LastSeen = change.Timestamp
		}
	}

	return nil
}