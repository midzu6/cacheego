package store

import (
	"sync"
	"time"
)

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
