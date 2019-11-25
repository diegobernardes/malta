package etcd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/etcd/clientv3"

	"malta/internal/service"
)

// Node implements the persistence logic for the node domain.
type Node struct {
	Client *Client
	kv     clientv3.KV
}

// Open setup the inner state.
func (n *Node) Open() {
	n.kv = clientv3.NewKV(n.Client.instance)
}

// Index is used to return the nodes.
func (n *Node) Index(ctx context.Context) ([]service.Node, error) {
	response, err := n.kv.Get(ctx, "/nodes/", clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nodes: %w", err)
	}

	result := make([]service.Node, len(response.Kvs))
	for i, kv := range response.Kvs {
		var n node
		if err := json.Unmarshal(kv.Value, &n); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the node: %w", err)
		}
		result[i] = service.Node(n)
	}

	return result, nil
}

// Create a new node.
func (n *Node) Create(ctx context.Context, node service.Node) error {
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(node); err != nil {
		return fmt.Errorf("error while serializing the node: %w", err)
	}
	key := fmt.Sprintf("/nodes/%s", node.ID)
	if _, err := n.kv.Put(ctx, key, buf.String(), clientv3.WithRev(0)); err != nil {
		return fmt.Errorf("error while persisting the new node: %w", err)
	}
	return nil
}

type node struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Port     uint              `json:"port"`
	Metadata map[string]string `json:"metadata"`
}
