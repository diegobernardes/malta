package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
)

// NodeCheck has the counters used to check if the node is health or not.
type NodeCheck struct {
	Client *Client

	stmtSelect    *sql.Stmt
	stmtIncrement *sql.Stmt
	stmtUpdate    *sql.Stmt
}

// Init internal state.
func (c *NodeCheck) Init() error {
	if c.Client == nil {
		return fmt.Errorf("missing client")
	}
	return nil
}

// Increment the counter value to the given node.
func (c *NodeCheck) Increment(ctx context.Context, id int) (int, error) {
	result, err := c.stmtIncrement.ExecContext(ctx, id, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to update: %w", err)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check if the row was updated: %w", err)
	}
	if affectedRows != 1 {
		return 0, fmt.Errorf("expected one row to be affected but '%d' was", affectedRows)
	}

	var count int
	if err := c.stmtSelect.QueryRow(id).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to fetch the count: %w", err)
	}
	return count, nil
}

// Update the node counter with the given value.
func (c *NodeCheck) Update(ctx context.Context, id, value int) error {
	result, err := c.stmtUpdate.ExecContext(ctx, value, id)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check if the row was updated: %w", err)
	}
	if affectedRows != 1 {
		return fmt.Errorf("expected one row to be affected but '%d' was", affectedRows)
	}
	return nil
}

func (c *NodeCheck) open() (err error) {
	querySelect := "SELECT count FROM node_check WHERE id = ?"
	c.stmtSelect, err = c.Client.instance.Prepare(querySelect)
	if err != nil {
		return fmt.Errorf("failed to create the select prepared statement: %w", err)
	}

	queryIncrement := `
		INSERT INTO node_check(id, count) VALUES (?, ?)
		ON CONFLICT (id) DO UPDATE SET count=count+1
	`
	c.stmtIncrement, err = c.Client.instance.Prepare(queryIncrement)
	if err != nil {
		return fmt.Errorf("failed to create the increment prepared statement: %w", err)
	}

	queryUpdate := "UPDATE node_check SET count = ? WHERE id = ?"
	c.stmtUpdate, err = c.Client.instance.Prepare(queryUpdate)
	if err != nil {
		return fmt.Errorf("failed to create the update prepared statement: %w", err)
	}

	return nil
}

func (c *NodeCheck) close() (err error) {
	if err := c.stmtSelect.Close(); err != nil {
		return fmt.Errorf("failed to close the select prepared statement: %w", err)
	}

	if err := c.stmtIncrement.Close(); err != nil {
		return fmt.Errorf("failed to close the increment prepared statement: %w", err)
	}

	if err := c.stmtUpdate.Close(); err != nil {
		return fmt.Errorf("failed to close the update prepared statement: %w", err)
	}

	return nil
}
