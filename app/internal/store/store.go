package store

import (
	"sync"
	"time"
)

type Store interface {
	Set(key string, val RedisValue, ttl time.Duration)
	Get(key string) (RedisValue, bool)
	Delete(key string)
}

type store struct {
	data map[string]RedisValue
	ttl  map[string]time.Time
	mu   sync.RWMutex
}
