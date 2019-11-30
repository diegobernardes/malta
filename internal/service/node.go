package service

import "time"

// Node representation.
type Node struct {
	ID        int
	Address   string
	Metadata  map[string]string
	TTL       time.Duration
	Active    bool
	CreatedAt time.Time
}
