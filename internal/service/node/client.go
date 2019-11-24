package node

import (
	"context"

	"malta/internal/service"
)

// Client implements the node bussiness logic.
type Client struct{}

// Index list the nodes.
func (*Client) Index(ctx context.Context) ([]service.Node, error) {
	return []service.Node{
		{
			ID:      "123",
			Address: "0.0.0.0",
			Port:    8081,
			Metadata: map[string]string{
				"machine": "456",
			},
		},
		{
			ID:      "124",
			Address: "0.0.0.0",
			Port:    8082,
			Metadata: map[string]string{
				"machine": "456",
			},
		},
		{
			ID:      "125",
			Address: "0.0.0.1",
			Port:    8083,
			Metadata: map[string]string{
				"machine": "457",
			},
		},
	}, nil
}
