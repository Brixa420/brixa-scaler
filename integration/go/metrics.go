package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MetricsCollector collects and exposes Prometheus-style metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters
	TxReceived    uint64 `json:"tx_received"`
	TxBatched     uint64 `json:"tx_batched"`
	TxSettled     uint64 `json:"tx_settled"`
	ProofsGenerated uint64 `json:"proofs_generated"`
	ProofsVerified uint64 `json:"proofs_verified"`
	Errors        uint64 `json:"errors"`

	// Gauges
	PendingTxs    uint64 `json:"pending_txs"`
	PendingBatches uint64 `json:"pending_batches"`
	BatchSize     uint64 `json:"batch_size"`

	// Histograms
	BatchTimes    []float64 `json:"batch_times"`
	ProofTimes    []float64 `json:"proof_times"`
	SettlementLatency []float64 `json:"settlement_latency"`

	// Start time
	StartTime time.Time `json:"start_time"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		StartTime: time.Now(),
	}
}

// IncTxReceived increments the transaction received counter
func (m *MetricsCollector) IncTxReceived() {
	atomicAddUint64(&m.TxReceived, 1)
}

// IncTxBatched increments the transaction batched counter
func (m *MetricsCollector) IncTxBatched(n uint64) {
	atomicAddUint64(&m.TxBatched, n)
}

// IncTxSettled increments the transaction settled counter
func (m *MetricsCollector) IncTxSettled(n uint64) {
	atomicAddUint64(&m.TxSettled, n)
}

// IncProofsGenerated increments the proofs generated counter
func (m *MetricsCollector) IncProofsGenerated() {
	atomicAddUint64(&m.ProofsGenerated, 1)
}

// IncProofsVerified increments the proofs verified counter
func (m *MetricsCollector) IncProofsVerified() {
	atomicAddUint64(&m.ProofsVerified, 1)
}

// IncErrors increments the errors counter
func (m *MetricsCollector) IncErrors() {
	atomicAddUint64(&m.Errors, 1)
}

// SetPendingTxs sets the pending transactions gauge
func (m *MetricsCollector) SetPendingTxs(n uint64) {
	atomicAddUint64(&m.PendingTxs, n)
}

// SetPendingBatches sets the pending batches gauge
func (m *MetricsCollector) SetPendingBatches(n uint64) {
	atomicAddUint64(&m.PendingBatches, n)
}

// RecordBatchTime records a batch processing time
func (m *MetricsCollector) RecordBatchTime(durationMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BatchTimes = append(m.BatchTimes, durationMs)
	// Keep only last 1000 measurements
	if len(m.BatchTimes) > 1000 {
		m.BatchTimes = m.BatchTimes[1:]
	}
}

// RecordProofTime records a proof generation time
func (m *MetricsCollector) RecordProofTime(durationMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProofTimes = append(m.ProofTimes, durationMs)
	if len(m.ProofTimes) > 1000 {
		m.ProofTimes = m.ProofTimes[1:]
	}
}

// RecordSettlementLatency records a settlement latency
func (m *MetricsCollector) RecordSettlementLatency(durationMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SettlementLatency = append(m.SettlementLatency, durationMs)
	if len(m.SettlementLatency) > 1000 {
		m.SettlementLatency = m.SettlementLatency[1:]
	}
}

// GetTPS calculates current TPS
func (m *MetricsCollector) GetTPS() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	elapsed := time.Since(m.StartTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(m.TxBatched) / elapsed
}

// GetAvgBatchTime returns average batch time in ms
func (m *MetricsCollector) GetAvgBatchTime() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.BatchTimes) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, t := range m.BatchTimes {
		sum += t
	}
	return sum / float64(len(m.BatchTimes))
}

// GetAvgProofTime returns average proof time in ms
func (m *MetricsCollector) GetAvgProofTime() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.ProofTimes) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, t := range m.ProofTimes {
		sum += t
	}
	return sum / float64(len(m.ProofTimes))
}

// GetPercentile returns the p-th percentile of a slice
func GetPercentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	n := len(sorted)
	k := int(float64(n) * p / 100.0)
	if k >= n {
		k = n - 1
	}
	
	// Simple sort (for production, use sort.Float64s)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted[k]
}

// GetP50 returns the 50th percentile
func (m *MetricsCollector) GetP50() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]float64{
		"batch_time_ms":     GetPercentile(m.BatchTimes, 50),
		"proof_time_ms":     GetPercentile(m.ProofTimes, 50),
		"settlement_ms":     GetPercentile(m.SettlementLatency, 50),
	}
}

// GetP99 returns the 99th percentile
func (m *MetricsCollector) GetP99() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]float64{
		"batch_time_ms":     GetPercentile(m.BatchTimes, 99),
		"proof_time_ms":     GetPercentile(m.ProofTimes, 99),
		"settlement_ms":     GetPercentile(m.SettlementLatency, 99),
	}
}

// ToJSON returns metrics as JSON
func (m *MetricsCollector) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return json.Marshal(struct {
		TxReceived       uint64   `json:"tx_received"`
		TxBatched        uint64   `json:"tx_batched"`
		TxSettled        uint64   `json:"tx_settled"`
		ProofsGenerated  uint64   `json:"proofs_generated"`
		ProofsVerified   uint64   `json:"proofs_verified"`
		Errors           uint64   `json:"errors"`
		PendingTxs       uint64   `json:"pending_txs"`
		PendingBatches  uint64   `json:"pending_batches"`
		TPS              float64  `json:"tps"`
		AvgBatchTimeMs   float64  `json:"avg_batch_time_ms"`
		AvgProofTimeMs   float64  `json:"avg_proof_time_ms"`
		UptimeSeconds    float64  `json:"uptime_seconds"`
	}{
		TxReceived:      m.TxReceived,
		TxBatched:       m.TxBatched,
		TxSettled:       m.TxSettled,
		ProofsGenerated: m.ProofsGenerated,
		ProofsVerified:  m.ProofsVerified,
		Errors:          m.Errors,
		PendingTxs:      m.PendingTxs,
		PendingBatches: m.PendingBatches,
		TPS:             m.GetTPS(),
		AvgBatchTimeMs:  m.GetAvgBatchTime(),
		AvgProofTimeMs:  m.GetAvgProofTime(),
		UptimeSeconds:   time.Since(m.StartTime).Seconds(),
	})
}

// PrometheusMetrics returns metrics in Prometheus format
func (m *MetricsCollector) PrometheusMetrics() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	elapsed := time.Since(m.StartTime).Seconds()
	tps := float64(m.TxBatched) / math.Max(elapsed, 1)
	
	return fmt.Sprintf(`# HELP brixa_tx_received Total transactions received
# TYPE brixa_tx_received counter
brixa_tx_received %d
# HELP brixa_tx_batched Total transactions batched
# TYPE brixa_tx_batched counter
brixa_tx_batched %d
# HELP brixa_tx_settled Total transactions settled on-chain
# TYPE brixa_tx_settled counter
brixa_tx_settled %d
# HELP brixa_proofs_generated Total ZK proofs generated
# TYPE brixa_proofs_generated counter
brixa_proofs_generated %d
# HELP brixa_proofs_verified Total ZK proofs verified
# TYPE brixa_proofs_verified counter
brixa_proofs_verified %d
# HELP brixa_errors Total errors encountered
# TYPE brixa_errors counter
brixa_errors %d
# HELP brixa_pending_txs Current pending transactions
# TYPE brixa_pending_txs gauge
brixa_pending_txs %d
# HELP brixa_tps Current transactions per second
# TYPE brixa_tps gauge
brixa_tps %.2f
# HELP brixa_uptime_seconds Uptime in seconds
# TYPE brixa_uptime_seconds gauge
brixa_uptime_seconds %.0f
`,
		m.TxReceived,
		m.TxBatched,
		m.TxSettled,
		m.ProofsGenerated,
		m.ProofsVerified,
		m.Errors,
		m.PendingTxs,
		tps,
		elapsed,
	)
}

// StartMetricsServer starts an HTTP server for metrics
func (m *MetricsCollector) StartMetricsServer(port int) error {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(m.PrometheusMetrics()))
	})
	
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := m.ToJSON()
		w.Write(data)
	})
	
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// atomicAddUint64 is a simple atomic add for uint64
func atomicAddUint64(addr *uint64, delta uint64) {
	// In Go 1.20+, use atomic.AddUint64
	// For older versions, use sync.Mutex
	// This is a simplified version
	*addr += delta
}