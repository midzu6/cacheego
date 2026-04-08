package store

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const tickDuration = 10 * time.Second

type Store interface {
	Set(key string, val RedisValue, ttl time.Duration)
	Get(key string) (RedisValue, bool)
	Delete(keys ...string) int64
}

type store struct {
	data   map[string]RedisValue
	expiry map[string]time.Time
	mu     sync.RWMutex
}

func NewStore() Store {
	return &store{
		data:   make(map[string]RedisValue),
		expiry: make(map[string]time.Time),
	}
}

func (s *store) Set(key string, val RedisValue, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = val

	if ttl > 0 {
		s.expiry[key] = time.Now().Add(ttl)
	} else {
		delete(s.expiry, key)
	}
}

func (s *store) Get(key string) (RedisValue, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, exist := s.data[key]

	if !exist {
		return nil, false
	}

	if timeExp, ok := s.expiry[key]; ok {
		if time.Now().After(timeExp) {
			delete(s.data, key)
			delete(s.expiry, key)
			return nil, false
		}
	}
	return val, true
}

func (s *store) Delete(keys ...string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deletedCount int64

	for _, key := range keys {
		if _, exist := s.data[key]; exist {
			delete(s.data, key)
			delete(s.expiry, key)
			deletedCount++
		}
	}
	return deletedCount
}

func (s *store) StartExpiry(ctx context.Context) {
	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted := s.DeleteExpired()
			if deleted > 0 {
				slog.Info("deleted expired keys", "count", deleted)
			}
		}
	}
}

func (s *store) DeleteExpired() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := time.Now()
	var count int64

	for key, val := range s.expiry {
		if t.After(val) {
			delete(s.data, key)
			delete(s.expiry, key)
			count++
		}
	}
	return count
}
