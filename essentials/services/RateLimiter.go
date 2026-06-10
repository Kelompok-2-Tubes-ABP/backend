package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// RateLimiter manages request rate limiting
type RateLimiter struct {
	mu sync.RWMutex
	requests     map[string][]time.Time
	maxRequests  int
	windowSize   time.Duration
	enabled      bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, windowSeconds int) *RateLimiter {
	return &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		windowSize:   time.Duration(windowSeconds) * time.Second,
		enabled:      true,
	}
}

// Allow checks if a request is allowed for the given key
func (r *RateLimiter) Allow(key string) (bool, int) {
	if !r.enabled {
		return true, r.maxRequests
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.windowSize)

	// Get existing requests and filter to window
	var validRequests []time.Time
	for _, t := range r.requests[key] {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check limit
	if len(validRequests) >= r.maxRequests {
		return false, r.maxRequests - len(validRequests)
	}

	// Add new request
	validRequests = append(validRequests, now)
	r.requests[key] = validRequests

	return true, r.maxRequests - len(validRequests)
}

// Cleanup removes old entries to prevent memory leaks
func (r *RateLimiter) Cleanup(maxAge time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for key, times := range r.requests {
		var valid []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(r.requests, key)
		} else {
			r.requests[key] = valid
		}
	}
}

// ResponseCache caches AI responses
type ResponseCache struct {
	mu       sync.RWMutex
	cache    map[string]cachedResponse
	maxAge   time.Duration
	maxSize  int
}

type cachedResponse struct {
	response  string
	timestamp time.Time
	hits      int
}

// NewResponseCache creates a new response cache
func NewResponseCache(maxAgeMinutes int, maxSize int) *ResponseCache {
	return &ResponseCache{
		cache:   make(map[string]cachedResponse),
		maxAge:  time.Duration(maxAgeMinutes) * time.Minute,
		maxSize: maxSize,
	}
}

// Get retrieves a cached response
func (c *ResponseCache) Get(query string) (string, bool) {
	key := c.hashQuery(query)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache[key]; ok {
		if time.Since(cached.timestamp) < c.maxAge {
			return cached.response, true
		}
	}

	return "", false
}

// Set stores a response in cache
func (c *ResponseCache) Set(query string, response string) {
	key := c.hashQuery(query)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if at capacity
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[key] = cachedResponse{
		response:  response,
		timestamp: time.Now(),
		hits:      0,
	}
}

// IncrementHits increases the hit count for a cached response
func (c *ResponseCache) IncrementHits(query string) {
	key := c.hashQuery(query)

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.cache[key]; ok {
		cached.hits++
		c.cache[key] = cached
	}
}

// hashQuery creates a hash key for the query
func (c *ResponseCache) hashQuery(query string) string {
	// Normalize query
	normalized := normalizeQuery(query)
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// normalizeQuery normalizes a query for consistent caching
func normalizeQuery(query string) string {
	// Convert to lowercase
	// Remove extra whitespace
	// This creates a consistent key for similar queries
	result := ""
	for i := 0; i < len(query); i++ {
		if query[i] != ' ' || (i > 0 && query[i-1] != ' ') {
			if query[i] >= 'A' && query[i] <= 'Z' {
				result += string(query[i] + 32) // to lowercase
			} else {
				result += string(query[i])
			}
		}
	}
	return result
}

// evictOldest removes the least recently used entry
func (c *ResponseCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range c.cache {
		if oldestKey == "" || cached.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.timestamp
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// Cleanup removes expired entries
func (c *ResponseCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.maxAge)
	for key, cached := range c.cache {
		if cached.timestamp.Before(cutoff) {
			delete(c.cache, key)
		}
	}
}

// Stats returns cache statistics
func (c *ResponseCache) Stats() (size int, avgHits float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	size = len(c.cache)
	totalHits := 0
	for _, cached := range c.cache {
		totalHits += cached.hits
	}

	if size > 0 {
		avgHits = float64(totalHits) / float64(size)
	}

	return
}

// TokenLimiter limits token usage
type TokenLimiter struct {
	mu          sync.RWMutex
	userTokens  map[string]tokenInfo
	maxTokens   int
	windowSize  time.Duration
}

type tokenInfo struct {
	tokens     int
	windowUsed time.Time
}

// NewTokenLimiter creates a new token limiter
func NewTokenLimiter(maxTokens int, windowMinutes int) *TokenLimiter {
	return &TokenLimiter{
		userTokens: make(map[string]tokenInfo),
		maxTokens:  maxTokens,
		windowSize: time.Duration(windowMinutes) * time.Minute,
	}
}

// CountTokens estimates token count for a string
func CountTokens(text string) int {
	// Rough estimate: ~4 characters per token for Indonesian/English mix
	// Groq uses byte-pair encoding, this is an approximation
	return len(text) / 4
}

// AllowTokens checks if tokens are allowed
func (t *TokenLimiter) AllowTokens(userID string, tokenCount int) (bool, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-t.windowSize)

	info, exists := t.userTokens[userID]
	if !exists || info.windowUsed.Before(windowStart) {
		// New window
		t.userTokens[userID] = tokenInfo{
			tokens:     tokenCount,
			windowUsed: now,
		}
		return true, t.maxTokens - tokenCount
	}

	// Check if within limit
	if info.tokens+tokenCount > t.maxTokens {
		return false, t.maxTokens - info.tokens
	}

	// Add tokens
	info.tokens += tokenCount
	t.userTokens[userID] = info

	return true, t.maxTokens - info.tokens
}

// ChatbotRateLimitManager manages all rate limiting for chatbot
type ChatbotRateLimitManager struct {
	requestLimiter *RateLimiter
	tokenLimiter   *TokenLimiter
	responseCache  *ResponseCache
}

// NewChatbotRateLimitManager creates a new rate limit manager
func NewChatbotRateLimitManager() *ChatbotRateLimitManager {
	return &ChatbotRateLimitManager{
		// 10 requests per minute per user
		requestLimiter: NewRateLimiter(10, 60),
		// 50,000 tokens per user per 10 minutes
		tokenLimiter: NewTokenLimiter(50000, 10),
		// Cache responses for 5 minutes, max 100 entries
		responseCache: NewResponseCache(5, 100),
	}
}

// AllowRequest checks if a request is allowed
func (m *ChatbotRateLimitManager) AllowRequest(userID string) (bool, string) {
	allowed, remaining := m.requestLimiter.Allow(userID)
	if !allowed {
		return false, fmt.Sprintf("Rate limit exceeded. Please wait before sending more messages. (Requests remaining: %d)", remaining)
	}
	return true, ""
}

// AllowTokens checks if token usage is allowed
func (m *ChatbotRateLimitManager) AllowTokens(userID string, message string) (bool, string, string) {
	// Check cache first
	if cached, ok := m.responseCache.Get(message); ok {
		m.responseCache.IncrementHits(message)
		return true, "", cached
	}

	// Estimate tokens
	tokenCount := CountTokens(message)
	allowed, remaining := m.tokenLimiter.AllowTokens(userID, tokenCount)
	if !allowed {
		return false, fmt.Sprintf("Token limit exceeded. Please wait before sending more messages. (Tokens remaining: %d)", remaining), ""
	}

	return true, "", ""
}

// CacheResponse stores a response in cache
func (m *ChatbotRateLimitManager) CacheResponse(message, response string) {
	m.responseCache.Set(message, response)
}

// GetCacheStats returns cache statistics
func (m *ChatbotRateLimitManager) GetCacheStats() (size int, avgHits float64) {
	return m.responseCache.Stats()
}

// StartCleanupRoutine starts a background cleanup routine
func (m *ChatbotRateLimitManager) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.requestLimiter.Cleanup(time.Hour)
			m.responseCache.Cleanup()
		}
	}()
}
