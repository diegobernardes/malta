package service

import "time"

// Node representation.
type Node struct {
	ID        int
	Address   string
	Metadata  map[string]string
	CreatedAt time.Time
}
