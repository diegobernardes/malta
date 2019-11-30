package handler

import (
	"time"

	"malta/internal/service"
)

type nodeViewCreate struct {
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata"`
}

type nodeViewList struct {
	Nodes []nodeView `json:"nodes"`
}

type nodeView struct {
	ID        int               `json:"id"`
	Address   string            `json:"address"`
	Metadata  map[string]string `json:"metadata"`
	TTL       string            `json:"ttl"`
	Active    bool              `json:"active"`
	CreatedAt string            `json:"createdAt"`
}

func toNodeView(n service.Node) nodeView {
	return nodeView{
		ID:        n.ID,
		Address:   n.Address,
		Metadata:  n.Metadata,
		TTL:       n.TTL.String(),
		Active:    n.Active,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
}

func toNodeViewList(nodes []service.Node) nodeViewList {
	if len(nodes) == 0 {
		return nodeViewList{Nodes: make([]nodeView, 0)}
	}
	result := nodeViewList{Nodes: make([]nodeView, len(nodes))}
	for i, n := range nodes {
		result.Nodes[i] = toNodeView(n)
	}
	return result
}

func toNode(nv nodeViewCreate) service.Node {
	return service.Node{
		Address:  nv.Address,
		Metadata: nv.Metadata,
	}
}
