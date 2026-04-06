package main

import (
	"log/slog"
	"time"
)

var tickDuration = time.Second * 10

type Entry struct {
	value     []byte
	expiresAt time.Time
}

func (s *Server) cleanupExpired() {
	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			deleted := s.deleteExpired()
			slog.Info("deleted expired keys", "count", deleted)
		}
	}
}

func (e Entry) isExpired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

func (s *Server) deleteExpired() int {
	count := 0

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.storage {
		if v.isExpired() {
			delete(s.storage, k)
			count++
		}
	}
	return count
}
