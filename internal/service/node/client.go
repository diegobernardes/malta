package node

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"malta/internal/database"
	"malta/internal/service"
)

// ClientRepository implements the node logic at the database layer.
type ClientRepository interface {
	Select(ctx context.Context) ([]service.Node, error)
	Insert(tx *sql.Tx, rawNode service.Node) error
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
func (c *Client) Create(ctx context.Context, node service.Node) (err error) {
	tx, err := c.Transaction.Begin(ctx, false, sql.LevelDefault)
	if err != nil {
		return fmt.Errorf("failed to create the transaction: %w", err)
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
	if err := c.Repository.Insert(tx, node); err != nil {
		return fmt.Errorf("failed to insert a new node: %w", err)
	}
	return nil
}
