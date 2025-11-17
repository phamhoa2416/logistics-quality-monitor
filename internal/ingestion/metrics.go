package ingestion

import (
	"sync"
	"time"
)

// IngestMetrics tracks ingestion performance
type IngestMetrics struct {
	MessagesReceived      int64
	MessagesProcessed     int64
	MessagesFailed        int64
	RecordsInserted       int64
	AlertsGenerated       int64
	LastProcessedAt       time.Time
	AverageProcessingTime time.Duration
	BufferSize            int
}

// MetricsTracker provides a goroutine-safe wrapper around IngestMetrics.
type MetricsTracker struct {
	mu        sync.RWMutex
	metrics   IngestMetrics
	listeners []func(IngestMetrics)
}

// NewMetricsTracker builds a new tracker with zeroed metrics.
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{}
}

// Update applies a mutation in a thread-safe way.
func (t *MetricsTracker) Update(fn func(*IngestMetrics)) {
	if fn == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	fn(&t.metrics)
	snapshot := t.metrics
	for _, listener := range t.listeners {
		listener(snapshot)
	}
}

// Snapshot returns a copy of the current metrics.
func (t *MetricsTracker) Snapshot() IngestMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.metrics
}

// Reset clears accumulated metrics.
func (t *MetricsTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics = IngestMetrics{}
}

// OnChange registers a callback invoked whenever metrics are updated.
func (t *MetricsTracker) OnChange(listener func(IngestMetrics)) {
	if listener == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.listeners = append(t.listeners, listener)
}
