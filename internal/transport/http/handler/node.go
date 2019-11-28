package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"malta/internal/service"
	"malta/internal/transport/http/shared"
)

type nodeRepository interface {
	Index(ctx context.Context) ([]service.Node, error)
	Create(ctx context.Context, node service.Node) (int, error)
}

// Node is the HTTP logic around the node business logic.
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

// Create a node.
func (n *Node) Create(w http.ResponseWriter, r *http.Request) {
	var nn node
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&nn); err != nil {
		n.Writer.Error(w, "failed parse request body", err, http.StatusInternalServerError)
		return
	}

	if _, err := n.Repository.Create(r.Context(), nn.origin()); err != nil {
		n.Writer.Error(w, "failed to create the the node", err, http.StatusInternalServerError)
		return
	}
}

type nodeList struct {
	Nodes []node `json:"nodes"`
}

type node struct {
	ID        int               `json:"id"`
	Address   string            `json:"address"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"createdAt"`
}

func (n node) origin() service.Node {
	return service.Node(n)
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
