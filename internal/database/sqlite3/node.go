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
		INSERT INTO node (address, metadata, created_at) VALUES (?, ?, ?)
	`
)

// nodeView represents the view of a node to the database layer.
type nodeView service.Node

func (n nodeView) insertArguments() ([]interface{}, error) {
	metadata, err := json.Marshal(n.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the node metadata: %w", err)
	}
	return []interface{}{n.Address, metadata, n.CreatedAt}, nil
}

// Node has the business logic around the database layer.
type Node struct {
	Client *Client

	stmtSelect *sql.Stmt
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
		err := rows.Scan(&node.ID, &node.Address, &metadata, &node.CreatedAt)
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

// Insert a node.
func (n *Node) Insert(tx *sql.Tx, rawNode service.Node) (int, error) {
	node := nodeView(rawNode)
	arguments, err := node.insertArguments()
	if err != nil {
		return 0, fmt.Errorf("failed to generate the insert arguments: %w", err)
	}

	result, err := tx.Exec(queryInsert, arguments...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert the node: %w", err)
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check if the row was inserted: %w", err)
	}
	if affectedRows != 1 {
		return 0, fmt.Errorf("expected one row to be affected but '%d' was", affectedRows)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch the insert id: %w", err)
	}
	return (int)(id), nil
}

func (n *Node) open() (err error) {
	querySelect := "SELECT id, address, metadata, created_at FROM node"
	n.stmtSelect, err = n.Client.instance.Prepare(querySelect)
	if err != nil {
		return fmt.Errorf("failed to create the select prepared statement: %w", err)
	}
	return nil
}

func (n *Node) close() (err error) {
	if err := n.stmtSelect.Close(); err != nil {
		return fmt.Errorf("failed to close the select prepared statement: %w", err)
	}
	return nil
}
