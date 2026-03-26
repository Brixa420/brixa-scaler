package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for BrixaScaler
type Config struct {
	Batching    BatchingConfig    `yaml:"batching"`
	ZK          ZKConfig           `yaml:"zk"`
	Settlement  SettlementConfig   `yaml:"settlement"`
	Persistence PersistenceConfig  `yaml:"persistence"`
	Retry       RetryConfig        `yaml:"retry"`
	Observability ObservabilityConfig `yaml:"observability"`
	Network     NetworkConfig      `yaml:"network"`
}

type BatchingConfig struct {
	MaxBatchSize    int `yaml:"max_batch_size"`
	BatchTimeoutMs  int `yaml:"batch_timeout_ms"`
	MaxPendingTxs  int `yaml:"max_pending_txs"`
}

type ZKConfig struct {
	CircuitName        string `yaml:"circuit_name"`
	Constraints        int    `yaml:"constraints"`
	ProvingTimeoutMs   int    `yaml:"proving_timeout_ms"`
	ParallelProofs     int    `yaml:"parallel_proofs"`
}

type SettlementConfig struct {
	TargetChain      string `yaml:"target_chain"`
	RpcURL           string `yaml:"rpc_url"`
	VerifierAddress  string `yaml:"verifier_address"`
	MaxSettlementGas int    `yaml:"max_settlement_gas"`
	GasPriceMultiplier float64 `yaml:"gas_price_multiplier"`
}

type PersistenceConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Backend     string `yaml:"backend"`  // leveldb, rocksdb, redis
	Path        string `yaml:"path"`
	SyncWrites  bool   `yaml:"sync_writes"`
	Compression bool   `yaml:"compression"`
}

type RetryConfig struct {
	MaxAttempts      int     `yaml:"max_attempts"`
	InitialDelayMs   int     `yaml:"initial_delay_ms"`
	MaxDelayMs       int     `yaml:"max_delay_ms"`
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`
	Jitter            bool    `yaml:"jitter"`
}

type ObservabilityConfig struct {
	MetricsEnabled  bool   `yaml:"metrics_enabled"`
	MetricsPort     int    `yaml:"metrics_port"`
	LogLevel        string `yaml:"log_level"`   // debug, info, warn, error
	LogFormat       string `yaml:"log_format"` // json, text
	TraceEnabled    bool   `yaml:"trace_enabled"`
	TraceEndpoint    string `yaml:"trace_endpoint"`
}

type NetworkConfig struct {
	RpcHost      string `yaml:"rpc_host"`
	RpcPort      int    `yaml:"rpc_port"`
	WsEnabled    bool   `yaml:"ws_enabled"`
	WsPort       int    `yaml:"ws_port"`
	MaxConnections int  `yaml:"max_connections"`
}

// LoadConfig loads configuration from YAML file with env variable substitution
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// expandEnvVars replaces ${VAR} patterns with environment variable values
func expandEnvVars(content string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[2 : len(match)-1]
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match
	})
}

// ExponentialBackoff calculates the next delay with optional jitter
func (r *RetryConfig) ExponentialBackoff(attempt int) time.Duration {
	delay := time.Duration(r.InitialDelayMs) * time.Millisecond
	
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * r.BackoffMultiplier)
		if delay > time.Duration(r.MaxDelayMs)*time.Millisecond {
			delay = time.Duration(r.MaxDelayMs) * time.Millisecond
		}
	}
	
	if r.Jitter {
		// Add +/- 25% jitter
		jitter := float64(delay) * 0.25
		delay = delay + time.Duration(jitter*(-1+2*float64(os.Getpid()%100)/100))
	}
	
	return delay
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Batching: BatchingConfig{
			MaxBatchSize:   1000,
			BatchTimeoutMs: 5000,
			MaxPendingTxs:  100000,
		},
		ZK: ZKConfig{
			CircuitName:      "batch_merkle",
			Constraints:      155000,
			ProvingTimeoutMs: 300000,
			ParallelProofs:   4,
		},
		Settlement: SettlementConfig{
			TargetChain:        "polygon",
			MaxSettlementGas:   500000,
			GasPriceMultiplier: 1.1,
		},
		Persistence: PersistenceConfig{
			Enabled:     true,
			Backend:     "leveldb",
			Path:        "./data/state.db",
			SyncWrites:  false,
			Compression: true,
		},
		Retry: RetryConfig{
			MaxAttempts:      5,
			InitialDelayMs:   1000,
			MaxDelayMs:       60000,
			BackoffMultiplier: 2.0,
			Jitter:            true,
		},
		Observability: ObservabilityConfig{
			MetricsEnabled: true,
			MetricsPort:    9090,
			LogLevel:       "info",
			LogFormat:      "json",
			TraceEnabled:   false,
		},
		Network: NetworkConfig{
			RpcHost:       "0.0.0.0",
			RpcPort:       8545,
			WsEnabled:     true,
			WsPort:        8546,
			MaxConnections: 1000,
		},
	}
}