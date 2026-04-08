package store

import (
	"sync"
	"time"
)

type Store interface {
}

type store struct {
	data map[string]RedisValue
	ttl  map[string]time.Time
	mu   sync.RWMutex
}
