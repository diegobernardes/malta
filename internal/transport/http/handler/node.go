package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"malta/internal/service"
	"malta/internal/transport/http/shared"
)

type nodeRepository interface {
	Index(ctx context.Context) ([]service.Node, error)
	FindOne(ctx context.Context, id string) (service.Node, error)
	Create(ctx context.Context, node service.Node) (service.Node, error)
}

// Node is the HTTP logic around the node business logic.
type Node struct {
	Repository      nodeRepository
	Writer          shared.Writer
	ResourceAddress func(service.Node) string
	ResourceID      func(*http.Request) string
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

	nodes := toNodeViewList(rawNodes)
	n.Writer.Response(w, nodes, http.StatusOK, nil)
}

// Show is used to show a single node.
func (n *Node) Show(w http.ResponseWriter, r *http.Request) {
	rawNode, err := n.Repository.FindOne(r.Context(), n.ResourceID(r))
	if err != nil {
		n.Writer.Error(w, "failed to fetch the node", err, http.StatusInternalServerError)
		return
	}

	node := toNodeView(rawNode)
	n.Writer.Response(w, node, http.StatusOK, nil)
}

// Create a node.
func (n *Node) Create(w http.ResponseWriter, r *http.Request) {
	var nv nodeViewCreate
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&nv); err != nil {
		n.Writer.Error(w, "failed parse request body", err, http.StatusInternalServerError)
		return
	}

	rawNode, err := n.Repository.Create(r.Context(), toNode(nv))
	if err != nil {
		n.Writer.Error(w, "failed to create the the node", err, http.StatusInternalServerError)
		return
	}
	node := toNodeView(rawNode)

	headers := http.Header{
		"Location": []string{
			n.ResourceAddress(rawNode),
		},
	}
	n.Writer.Response(w, node, http.StatusCreated, headers)
}
