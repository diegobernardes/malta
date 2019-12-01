package sqlite3

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"malta/internal/service"
)

const (
	queryInsert = `
		INSERT INTO node (address, metadata, ttl, active, created_at) VALUES (?, ?, ?, ?, ?)
	`
)

// Node has the business logic around the database layer.
type Node struct {
	Client *Client

	stmtSelect    *sql.Stmt
	stmtSelectOne *sql.Stmt
	stmtUpdate    *sql.Stmt
}

// Init internal state.
func (n *Node) Init() error {
	if n.Client == nil {
		return fmt.Errorf("missing client")
	}
	return nil
}

// Select return a list of nodes.
func (n *Node) Select(ctx context.Context) ([]service.Node, error) {
	rows, err := n.stmtSelect.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute que query: %w", err)
	}
	defer rows.Close()

	var nodes []service.Node
	for rows.Next() {
		var (
			node     service.Node
			metadata []byte
		)
		err := rows.Scan(&node.ID, &node.Address, &metadata, &node.TTL, &node.Active, &node.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the rows: %w", err)
		}

		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &node.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to process the rows: %w", err)
	}
	return nodes, nil
}

// SelectOne is used to get a single node.
func (n *Node) SelectOne(ctx context.Context, id string) (service.Node, error) {
	var (
		node     service.Node
		metadata []byte
		row      = n.stmtSelectOne.QueryRowContext(ctx, id)
	)
	err := row.Scan(&node.ID, &node.Address, &metadata, &node.TTL, &node.Active, &node.CreatedAt)
	if err != nil {
		return service.Node{}, fmt.Errorf("failed to parse the rows: %w", err)
	}

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &node.Metadata); err != nil {
			return service.Node{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	return node, nil
}

// Insert a node.
func (n *Node) Insert(tx *sql.Tx, node service.Node) (service.Node, error) {
	arguments, err := nodeInsertArguments(node)
	if err != nil {
		return service.Node{}, fmt.Errorf("failed to generate the insert arguments: %w", err)
	}

	result, err := tx.Exec(queryInsert, arguments...)
	if err != nil {
		return service.Node{}, fmt.Errorf("failed to insert the node: %w", err)
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return service.Node{}, fmt.Errorf("failed to check if the row was inserted: %w", err)
	}
	if affectedRows != 1 {
		return service.Node{}, fmt.Errorf("expected one row to be affected but '%d' was", affectedRows)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return service.Node{}, fmt.Errorf("failed to fetch the insert id: %w", err)
	}
	node.ID = (int)(id)
	return node, nil
}

// Update a given node.
func (n *Node) Update(ctx context.Context, node service.Node) error {
	arguments, err := nodeInsertArguments(node)
	if err != nil {
		return fmt.Errorf("failed to generate the insert arguments: %w", err)
	}
	arguments = append(arguments, node.ID)

	result, err := n.stmtUpdate.ExecContext(ctx, arguments...)
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

func (n *Node) open() (err error) {
	querySelect := `SELECT id, address, metadata, ttl, active, created_at
										FROM node
									 WHERE active = true
								ORDER BY created_at`
	n.stmtSelect, err = n.Client.instance.Prepare(querySelect)
	if err != nil {
		return fmt.Errorf("failed to create the select prepared statement: %w", err)
	}

	querySelectOne := "SELECT id, address, metadata, ttl, active, created_at FROM node WHERE id = ?"
	n.stmtSelectOne, err = n.Client.instance.Prepare(querySelectOne)
	if err != nil {
		return fmt.Errorf("failed to create the select one prepared statement: %w", err)
	}

	queryUpdate := `UPDATE node
									   SET address = ?, metadata = ?, ttl = ?, active = ?, created_at = ?
									 WHERE id = ?`
	n.stmtUpdate, err = n.Client.instance.Prepare(queryUpdate)
	if err != nil {
		return fmt.Errorf("failed to create the update prepared statement: %w", err)
	}

	return nil
}

func (n *Node) close() (err error) {
	if err := n.stmtSelect.Close(); err != nil {
		return fmt.Errorf("failed to close the select prepared statement: %w", err)
	}

	if err := n.stmtSelectOne.Close(); err != nil {
		return fmt.Errorf("failed to close the select one prepared statement: %w", err)
	}

	if err := n.stmtUpdate.Close(); err != nil {
		return fmt.Errorf("failed to close the update prepared statement: %w", err)
	}
	return nil
}

func nodeInsertArguments(n service.Node) ([]interface{}, error) {
	metadata, err := json.Marshal(n.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the node metadata: %w", err)
	}
	return []interface{}{n.Address, metadata, n.TTL.Nanoseconds(), n.Active, n.CreatedAt}, nil
}
