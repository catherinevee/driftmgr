package timeline

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	EventTypeDiscovery   EventType = "discovery"
	EventTypeDrift       EventType = "drift"
	EventTypeAdoption    EventType = "adoption"
	EventTypeConflict    EventType = "conflict"
	EventTypeRemediation EventType = "remediation"
	EventTypeStateChange EventType = "state_change"
)

type EventSeverity string

const (
	SeverityInfo    EventSeverity = "info"
	SeverityWarning EventSeverity = "warning"
	SeverityError   EventSeverity = "error"
)

type TimelineEvent struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Severity    EventSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	StateFile   string                 `json:"stateFile,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

type Timeline struct {
	events    []TimelineEvent
	mu        sync.RWMutex
	maxEvents int
	listeners []chan TimelineEvent
}

func NewTimeline(maxEvents int) *Timeline {
	if maxEvents <= 0 {
		maxEvents = 1000
	}
	return &Timeline{
		events:    make([]TimelineEvent, 0, maxEvents),
		maxEvents: maxEvents,
		listeners: make([]chan TimelineEvent, 0),
	}
}

func (t *Timeline) AddEvent(event TimelineEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.ID == "" {
		event.ID = fmt.Sprintf("event-%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	t.events = append([]TimelineEvent{event}, t.events...)

	if len(t.events) > t.maxEvents {
		t.events = t.events[:t.maxEvents]
	}

	for _, listener := range t.listeners {
		select {
		case listener <- event:
		default:
		}
	}
}

func (t *Timeline) GetEvents(filter TimelineFilter) []TimelineEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	filtered := make([]TimelineEvent, 0)
	
	for _, event := range t.events {
		if t.matchesFilter(event, filter) {
			filtered = append(filtered, event)
			if filter.Limit > 0 && len(filtered) >= filter.Limit {
				break
			}
		}
	}

	return filtered
}

func (t *Timeline) matchesFilter(event TimelineEvent, filter TimelineFilter) bool {
	if !filter.StartTime.IsZero() && event.Timestamp.Before(filter.StartTime) {
		return false
	}
	
	if !filter.EndTime.IsZero() && event.Timestamp.After(filter.EndTime) {
		return false
	}
	
	if filter.Type != "" && event.Type != filter.Type {
		return false
	}
	
	if filter.Severity != "" && event.Severity != filter.Severity {
		return false
	}
	
	if filter.StateFile != "" && event.StateFile != filter.StateFile {
		return false
	}
	
	if filter.Provider != "" && event.Provider != filter.Provider {
		return false
	}
	
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, eventTag := range event.Tags {
				if filterTag == eventTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}
	
	return true
}

func (t *Timeline) Subscribe() <-chan TimelineEvent {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	ch := make(chan TimelineEvent, 100)
	t.listeners = append(t.listeners, ch)
	return ch
}

func (t *Timeline) Unsubscribe(ch <-chan TimelineEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	for i, listener := range t.listeners {
		if listener == ch {
			close(listener)
			t.listeners = append(t.listeners[:i], t.listeners[i+1:]...)
			break
		}
	}
}

func (t *Timeline) GetStatistics(duration time.Duration) TimelineStatistics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	stats := TimelineStatistics{
		EventCounts:    make(map[EventType]int),
		SeverityCounts: make(map[EventSeverity]int),
		ProviderCounts: make(map[string]int),
	}
	
	cutoff := time.Now().Add(-duration)
	
	for _, event := range t.events {
		if event.Timestamp.Before(cutoff) {
			break
		}
		
		stats.TotalEvents++
		stats.EventCounts[event.Type]++
		stats.SeverityCounts[event.Severity]++
		
		if event.Provider != "" {
			stats.ProviderCounts[event.Provider]++
		}
		
		if stats.FirstEvent.IsZero() || event.Timestamp.Before(stats.FirstEvent) {
			stats.FirstEvent = event.Timestamp
		}
		
		if event.Timestamp.After(stats.LastEvent) {
			stats.LastEvent = event.Timestamp
		}
	}
	
	return stats
}

func (t *Timeline) Export(filter TimelineFilter) ([]byte, error) {
	events := t.GetEvents(filter)
	return json.MarshalIndent(events, "", "  ")
}

func (t *Timeline) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = make([]TimelineEvent, 0, t.maxEvents)
}

type TimelineFilter struct {
	StartTime time.Time
	EndTime   time.Time
	Type      EventType
	Severity  EventSeverity
	StateFile string
	Provider  string
	Tags      []string
	Limit     int
}

type TimelineStatistics struct {
	TotalEvents    int
	EventCounts    map[EventType]int
	SeverityCounts map[EventSeverity]int
	ProviderCounts map[string]int
	FirstEvent     time.Time
	LastEvent      time.Time
}

type StateHistory struct {
	StateFile string            `json:"stateFile"`
	Versions  []StateVersion    `json:"versions"`
	Changes   []StateChange     `json:"changes"`
	mu        sync.RWMutex
}

type StateVersion struct {
	Version       int                    `json:"version"`
	Timestamp     time.Time              `json:"timestamp"`
	ResourceCount int                    `json:"resourceCount"`
	Hash          string                 `json:"hash"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type StateChange struct {
	Timestamp   time.Time   `json:"timestamp"`
	ChangeType  string      `json:"changeType"`
	Resource    string      `json:"resource"`
	OldValue    interface{} `json:"oldValue,omitempty"`
	NewValue    interface{} `json:"newValue,omitempty"`
	Description string      `json:"description"`
}

func NewStateHistory(stateFile string) *StateHistory {
	return &StateHistory{
		StateFile: stateFile,
		Versions:  make([]StateVersion, 0),
		Changes:   make([]StateChange, 0),
	}
}

func (sh *StateHistory) AddVersion(version StateVersion) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	
	if version.Timestamp.IsZero() {
		version.Timestamp = time.Now()
	}
	
	sh.Versions = append(sh.Versions, version)
	
	if len(sh.Versions) > 100 {
		sh.Versions = sh.Versions[len(sh.Versions)-100:]
	}
}

func (sh *StateHistory) AddChange(change StateChange) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	
	if change.Timestamp.IsZero() {
		change.Timestamp = time.Now()
	}
	
	sh.Changes = append([]StateChange{change}, sh.Changes...)
	
	if len(sh.Changes) > 500 {
		sh.Changes = sh.Changes[:500]
	}
}

func (sh *StateHistory) GetVersions(limit int) []StateVersion {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	if limit <= 0 || limit > len(sh.Versions) {
		limit = len(sh.Versions)
	}
	
	result := make([]StateVersion, limit)
	copy(result, sh.Versions[len(sh.Versions)-limit:])
	
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	
	return result
}

func (sh *StateHistory) GetChanges(since time.Time, limit int) []StateChange {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	filtered := make([]StateChange, 0)
	
	for _, change := range sh.Changes {
		if !since.IsZero() && change.Timestamp.Before(since) {
			break
		}
		filtered = append(filtered, change)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	
	return filtered
}

func (sh *StateHistory) GetLatestVersion() *StateVersion {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	if len(sh.Versions) == 0 {
		return nil
	}
	
	return &sh.Versions[len(sh.Versions)-1]
}

func (sh *StateHistory) CompareVersions(v1, v2 int) []StateChange {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	changes := make([]StateChange, 0)
	
	var version1, version2 *StateVersion
	for _, v := range sh.Versions {
		if v.Version == v1 {
			version1 = &v
		}
		if v.Version == v2 {
			version2 = &v
		}
	}
	
	if version1 == nil || version2 == nil {
		return changes
	}
	
	startTime := version1.Timestamp
	endTime := version2.Timestamp
	if startTime.After(endTime) {
		startTime, endTime = endTime, startTime
	}
	
	for _, change := range sh.Changes {
		if change.Timestamp.After(startTime) && change.Timestamp.Before(endTime) {
			changes = append(changes, change)
		}
	}
	
	return changes
}