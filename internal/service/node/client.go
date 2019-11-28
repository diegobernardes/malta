package node

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"malta/internal/database"
	"malta/internal/service"
)

// ClientRepository implements the node logic at the database layer.
type ClientRepository interface {
	Select(ctx context.Context) ([]service.Node, error)
	Insert(tx *sql.Tx, rawNode service.Node) (int, error)
}

// ClientNotification implements the node logic to notify whenever a node is created.
type ClientNotification interface {
	Add(node service.Node)
}

// Client implements the node business logic.
type Client struct {
	Repository         ClientRepository
	Notification       ClientNotification
	Transaction        database.Transaction
	TransactionHandler func(*sql.Tx, error) error
}

// Index list the nodes.
func (c *Client) Index(ctx context.Context) ([]service.Node, error) {
	return c.Repository.Select(ctx)
}

// Create a node.
func (c *Client) Create(ctx context.Context, node service.Node) (_ int, err error) {
	if _, err := url.Parse(node.Address); err != nil {
		return 0, fmt.Errorf("invalid address: %w", err)
	}

	tx, err := c.Transaction.Begin(ctx, false, sql.LevelDefault)
	if err != nil {
		return 0, fmt.Errorf("failed to create the transaction: %w", err)
	}
	defer func() {
		err = c.TransactionHandler(tx, err)
		if err != nil {
			return
		}
		c.Notification.Add(node)
	}()

	if node.Metadata == nil {
		node.Metadata = make(map[string]string)
	}
	node.CreatedAt = time.Now().UTC()

	node.ID, err = c.Repository.Insert(tx, node)
	if err != nil {
		return 0, fmt.Errorf("failed to insert a new node: %w", err)
	}
	return node.ID, nil
}
