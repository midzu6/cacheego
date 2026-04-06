package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrWrongNumberOfArgument = errors.New("ERR wrong number of arguments")
var ErrKeyNotExists = errors.New("ERR key no exists")
var ErrKeyIsExpired = errors.New("ERR key is expired")

func (s *Server) cmdEcho(args [][]byte) ([]byte, error) {
	if len(args) == 0 {
		return nil, ErrWrongNumberOfArgument
	}
	return args[0], nil
}

func (s *Server) cmdSet(args [][]byte) error {
	if len(args) < 2 {
		return ErrWrongNumberOfArgument
	}
	var expiresAt time.Time

	for i := 2; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return ErrWrongNumberOfArgument
		}
		opt := strings.ToUpper(string(args[i]))
		t, convErr := strconv.Atoi(string(args[i+1]))
		if convErr != nil {
			return fmt.Errorf("ERR value is not an integer or out of range")
		}
		switch opt {
		case "EX":
			expiresAt = time.Now().Add(time.Duration(t) * time.Second)
		case "PX":
			expiresAt = time.Now().Add(time.Duration(t) * time.Millisecond)
		default:
			return fmt.Errorf("ERR unknown option '%s'", opt)
		}
	}
	s.mu.Lock()
	s.storage[string(args[0])] = Entry{
		value:     args[1],
		expiresAt: expiresAt,
	}
	s.mu.Unlock()

	return nil
}

func (s *Server) cmdGet(args [][]byte) ([]byte, error) {
	if len(args) < 1 {
		return nil, ErrWrongNumberOfArgument
	}
	s.mu.RLock()
	entr, ok := s.storage[string(args[0])]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrKeyNotExists
	} else if !entr.expiresAt.IsZero() && time.Now().After(entr.expiresAt) {
		return nil, ErrKeyIsExpired
	} else {
		return entr.value, nil
	}
}
