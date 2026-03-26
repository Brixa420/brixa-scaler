package main

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SecurityConfig holds security settings
type SecurityConfig struct {
	APIKey              string
	RateLimitPerSecond  int
	RateLimitBurst      int
	MaxRequestSize     int64
	MaxGasPriceGwei     uint64
	EnableHTTPS         bool
	CORSOrigins         []string
}

// LoadSecurityConfig loads security config from environment
func LoadSecurityConfig() SecurityConfig {
	return SecurityConfig{
		APIKey:             os.Getenv("API_KEY"),
		RateLimitPerSecond: getEnvInt("RATE_LIMIT_PER_SECOND", 10),
		RateLimitBurst:     getEnvInt("RATE_LIMIT_BURST", 20),
		MaxRequestSize:     getEnvInt64("MAX_REQUEST_SIZE", 1024*1024), // 1MB
		MaxGasPriceGwei:    uint64(getEnvInt("MAX_GAS_PRICE_GWEI", 100)),
		EnableHTTPS:        getEnvBool("ENABLE_HTTPS", false),
		CORSOrigins:        strings.Split(os.Getenv("CORS_ORIGINS"), ","),
	}
}

func getEnvInt64(key string, def int64) int64 {
	var val int64
	fmt.Sscanf(os.Getenv(key), "%d", &val)
	if val == 0 {
		return def
	}
	return val
}

// APIKeyMiddleware validates API key
func APIKeyMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		
		provided := r.Header.Get("X-API-Key")
		if provided == "" {
			provided = r.URL.Query().Get("api_key")
		}
		
		if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimitMiddleware limits request size
func RequestSizeLimitMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxSize {
				http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware
func CORSMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(origins) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			
			origin := r.Header.Get("Origin")
			for _, allowed := range origins {
				if origin == allowed || allowed == "*" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
					break
				}
			}
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NonceTracker tracks nonces per address
type NonceTracker struct {
	nonces map[string]uint64
	mu     sync.Mutex
}

func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		nonces: make(map[string]uint64),
	}
}

func (nt *NonceTracker) GetNonce(address string) uint64 {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	return nt.nonces[address]
}

func (nt *NonceTracker) SetNonce(address string, nonce uint64) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.nonces[address] = nonce
}

func (nt *NonceTracker) IncrementNonce(address string) uint64 {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.nonces[address]++
	return nt.nonces[address]
}

// ValidatePrivateKey validates key format
func ValidatePrivateKey(key string) error {
	if key == "" {
		return fmt.Errorf("private key is empty")
	}
	
	// Remove 0x prefix if present
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}
	
	// Validate hex length (32 bytes = 64 hex chars)
	if len(key) != 64 {
		return fmt.Errorf("invalid private key length: %d (expected 64)", len(key))
	}
	
	_, err := hex.DecodeString(key)
	if err != nil {
		return fmt.Errorf("invalid private key format: %v", err)
	}
	
	return nil
}
