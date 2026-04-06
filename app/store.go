package main

import "time"

type Entry struct {
	value     []byte
	expiresAt time.Time
}
