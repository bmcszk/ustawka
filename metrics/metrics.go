package metrics

import (
	"sync/atomic"
)

var (
	// API calls counter
	apiCalls uint64

	// Sejm API calls counter
	sejmAPICalls uint64

	// Cache hits counter
	cacheHits uint64

	// Cache misses counter
	cacheMisses uint64
)

// IncrementAPI calls counter
func IncrementAPI() {
	atomic.AddUint64(&apiCalls, 1)
}

// IncrementSejmAPI calls counter
func IncrementSejmAPI() {
	atomic.AddUint64(&sejmAPICalls, 1)
}

// IncrementCacheHit increments cache hits counter
func IncrementCacheHit() {
	atomic.AddUint64(&cacheHits, 1)
}

// IncrementCacheMiss increments cache misses counter
func IncrementCacheMiss() {
	atomic.AddUint64(&cacheMisses, 1)
}

// GetMetrics returns current metrics values
func GetMetrics() map[string]uint64 {
	return map[string]uint64{
		"api_calls":      atomic.LoadUint64(&apiCalls),
		"sejm_api_calls": atomic.LoadUint64(&sejmAPICalls),
		"cache_hits":     atomic.LoadUint64(&cacheHits),
		"cache_misses":   atomic.LoadUint64(&cacheMisses),
	}
}
