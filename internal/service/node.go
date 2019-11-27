package service

import "time"

// Node representation.
type Node struct {
	ID        int
	Address   string
	Port      uint
	Metadata  map[string]string
	CreatedAt time.Time
}
