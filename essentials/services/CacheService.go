package services

import (
	"errors"
	"sync"
	"time"
)

type CacheService struct {
	mu        sync.RWMutex
	data      map[string]cacheItem
	syncQueue []SyncQueueItem
}

type cacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

var globalCache *CacheService
var cacheOnce sync.Once

func GetCacheService() *CacheService {
	cacheOnce.Do(func() {
		globalCache = &CacheService{
			data:      make(map[string]cacheItem),
			syncQueue: make([]SyncQueueItem, 0),
		}
	})
	return globalCache
}

func (s *CacheService) Set(key string, value interface{}, expiration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = cacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(expiration),
	}
}

func (s *CacheService) Get(key string, dest interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.data[key]
	if !exists {
		return errors.New("cache miss")
	}

	if time.Now().After(item.ExpiresAt) {
		delete(s.data, key)
		return errors.New("cache expired")
	}

	return nil
}

func (s *CacheService) GetValue(key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.data[key]
	if !exists {
		return nil, errors.New("cache miss")
	}

	if time.Now().After(item.ExpiresAt) {
		delete(s.data, key)
		return nil, errors.New("cache expired")
	}

	return item.Value, nil
}

func (s *CacheService) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *CacheService) SetUserData(userID string, dataType string, data interface{}) {
	key := "user:" + userID + ":" + dataType
	s.Set(key, data, 1*time.Hour)
}

func (s *CacheService) GetUserData(userID string, dataType string) (interface{}, error) {
	key := "user:" + userID + ":" + dataType
	return s.GetValue(key)
}

func (s *CacheService) InvalidateUserData(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key := range s.data {
		if len(key) > 5 && key[:5] == "user:"+userID {
			delete(s.data, key)
		}
	}
}

func (s *CacheService) ClearExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, item := range s.data {
		if now.After(item.ExpiresAt) {
			delete(s.data, key)
		}
	}
}

type SyncQueueItem struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

func (s *CacheService) AddToSyncQueue(item SyncQueueItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncQueue = append(s.syncQueue, item)
}

func (s *CacheService) GetSyncQueue() []SyncQueueItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SyncQueueItem, len(s.syncQueue))
	copy(result, s.syncQueue)
	return result
}

func (s *CacheService) ClearSyncQueue() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncQueue = make([]SyncQueueItem, 0)
}

type OfflineCache struct {
	UserID       string    `json:"user_id"`
	LastSyncTime time.Time `json:"last_sync_time"`
	CachedTypes  []string  `json:"cached_types"`
}

var syncMeta = make(map[string]OfflineCache)
var metaMu sync.RWMutex

func (s *CacheService) RecordSync(userID string, dataType string) {
	metaMu.Lock()
	defer metaMu.Unlock()

	cache := OfflineCache{
		UserID:       userID,
		LastSyncTime: time.Now(),
		CachedTypes:  []string{dataType},
	}
	syncMeta[userID] = cache
}

func (s *CacheService) GetLastSyncTime(userID string) time.Time {
	metaMu.RLock()
	defer metaMu.RUnlock()

	if cache, exists := syncMeta[userID]; exists {
		return cache.LastSyncTime
	}
	return time.Time{}
}

func (s *CacheService) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cached_items": len(s.data),
		"sync_queue":   len(s.syncQueue),
	}
}

func (s *CacheService) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]cacheItem)
	s.syncQueue = make([]SyncQueueItem, 0)
}
