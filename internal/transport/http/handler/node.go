package handler

import (
	"context"
	"fmt"
	"net/http"

	"malta/internal/service"
	"malta/internal/transport/http/shared"
)

type nodeRepository interface {
	Index(ctx context.Context) ([]service.Node, error)
}

// Node is the HTTP logic around the node bussiness logic.
type Node struct {
	Repository nodeRepository
	Writer     shared.Writer
}

// Init internal state.
func (n *Node) Init() error {
	if n.Repository == nil {
		return fmt.Errorf("repository can't be nil")
	}
	return nil
}

// Index is used to list the nodes.
func (n *Node) Index(w http.ResponseWriter, r *http.Request) {
	rawNodes, err := n.Repository.Index(r.Context())
	if err != nil {
		n.Writer.Error(w, "failed to fetch the nodes", err, http.StatusInternalServerError)
		return
	}

	nodes := toNodeList(rawNodes)
	n.Writer.Response(w, nodes, http.StatusOK, nil)
}

type nodeList struct {
	Nodes []node `json:"nodes"`
}

type node struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Port     uint              `json:"port"`
	Metadata map[string]string `json:"metadata"`
}

func toNodeList(nodes []service.Node) nodeList {
	if len(nodes) == 0 {
		return nodeList{Nodes: make([]node, 0)}
	}
	result := nodeList{Nodes: make([]node, len(nodes))}
	for i, n := range nodes {
		result.Nodes[i] = node(n)
	}
	return result
}
