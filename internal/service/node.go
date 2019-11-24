package service

// Node representation.
type Node struct {
	ID       string
	Address  string
	Port     uint
	Metadata map[string]string
}
