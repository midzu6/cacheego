package store

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const tickDuration = 10 * time.Second

type Store interface {
	Set(key string, val RedisValue, ttl time.Duration)
	Get(key string) (RedisValue, bool)
	Delete(keys ...string) int64
	StartExpiry(ctx context.Context)
	RPush(key string, values ...string) (int64, error)
	LRange(key string, start, stop int) ([]string, error)
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
			deleted := s.deleteExpired()
			if deleted > 0 {
				slog.Info("deleted expired keys", "count", deleted)
			}
		}
	}
}

func (s *store) deleteExpired() int64 {
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

func (s *store) RPush(key string, values ...string) (int64, error) {
	return s.push(key, values, false)
}

func (s *store) LPush(key string, values ...string) (int64, error) {
	return s.push(key, values, true)
}

func (s *store) push(key string, values []string, left bool) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lv ListValue
	val, exist := s.data[key]
	if !exist {
		lv = NewListValue()
	} else {
		var ok bool
		lv, ok = val.(ListValue)
		if !ok {
			return 0, errors.New("ERR WRONG TYPE Operation against a key holding the wrong kind of value")
		}
	}
	if left {
		for _, v := range values {
			lv.Data.PushFront(v)
		}
	} else {
		for _, v := range values {
			lv.Data.PushBack(v)
		}
	}
	s.data[key] = lv
	return lv.Size(), nil
}

func (s *store) LRange(key string, start, stop int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, exists := s.data[key]
	if !exists {
		return []string{}, nil
	}

	lv, ok := val.(ListValue)
	if !ok {
		return nil, errors.New("ERR WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	length := int(lv.Size())

	if start > stop || start >= length {
		return []string{}, nil
	}

	if stop >= length {
		stop = length - 1
	}

	result := make([]string, 0, stop-start+1)

	i := 0
	for e := lv.Data.Front(); e != nil; e = e.Next() {
		if i > stop {
			break
		}
		if i >= start {
			result = append(result, e.Value.(string))
		}
		i++
	}

	return result, nil
}
