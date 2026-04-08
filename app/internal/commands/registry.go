package commands

import (
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/internal/parser"
	"github.com/codecrafters-io/redis-starter-go/app/internal/store"
)

type Command interface {
	Name() string
	Execute(args []parser.Value, st store.Store) (parser.Value, error)
}

type Registry struct {
	mu  sync.RWMutex
	cmd map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{cmd: make(map[string]Command)}
}
func (r *Registry) Register(c Command) {
	r.mu.Lock()
	r.cmd[c.Name()] = c
	r.mu.Unlock()
}

func (r *Registry) Get(name string) (Command, bool) {
	r.mu.RLock()
	c, ok := r.cmd[name]
	r.mu.RUnlock()
	return c, ok
}
