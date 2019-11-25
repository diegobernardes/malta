package node

import (
	"context"

	"malta/internal/service"
)

type ClientRepository interface {
	Index(ctx context.Context) ([]service.Node, error)
	Create(ctx context.Context, node service.Node) error
}

// Client implements the node bussiness logic.
type Client struct {
	Repository ClientRepository
}

// Index list the nodes.
func (c *Client) Index(ctx context.Context) ([]service.Node, error) {
	return c.Repository.Index(ctx)
}

// Create a node.
func (c *Client) Create(ctx context.Context, node service.Node) error {
	return c.Repository.Create(ctx, node)
}
