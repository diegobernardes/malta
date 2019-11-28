package node

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"malta/internal/service"
)

// HealthConfigRepository load all the nodes.
type HealthConfigRepository interface {
	Select(ctx context.Context) ([]service.Node, error)
}

// HealthConfig used to setup the health internal state.
type HealthConfig struct {
	// Interval used to check the nodes health.
	Interval time.Duration

	// Used to fetch the nodes.
	Repository HealthConfigRepository

	// Concurrency used to check the nodes.
	Concurrency int

	HTTPClient *http.Client
	Logger     zerolog.Logger
}

// Health is used to track the health of the nodes.
type Health struct {
	Config HealthConfig

	add       chan service.Node
	ctx       context.Context
	ctxCancel func()
	nodes     []service.Node
	wg        sync.WaitGroup
}

// Start the process.
func (h *Health) Start() error {
	h.add = make(chan service.Node)
	var err error
	h.nodes, err = h.Config.Repository.Select(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %w", err)
	}
	h.wg.Add(1)
	go h.process()
	return nil
}

// Stop the process.
func (h *Health) Stop() {
	h.ctxCancel()
	h.wg.Wait()
}

// Add the node to be checked.
func (h *Health) Add(node service.Node) {
	go func() { h.add <- node }()
}

func (h *Health) process() {
	defer h.wg.Done()
	h.ctx, h.ctxCancel = context.WithCancel(context.Background())

	for {
		select {
		case node := <-h.add:
			h.Config.Logger.Debug().Int("nodeID", node.ID).Msg("received node creation notification")
			h.nodes = append(h.nodes, node)
			continue
		case <-h.ctx.Done():
			return
		case <-time.After(h.Config.Interval):
		}

		g, gctx := errgroup.WithContext(h.ctx)
		ratelimit := make(chan struct{}, h.Config.Concurrency)
		for _, node := range h.nodes {
			ratelimit <- struct{}{}
			g.Go(h.check(gctx, ratelimit, node))
		}
		g.Wait() // nolinter: errcheck
	}
}

func (h *Health) check(ctx context.Context, rl <-chan struct{}, node service.Node) func() error {
	return func() error {
		defer func() { <-rl }()

		address := fmt.Sprintf("%s/health", node.Address)
		req, err := http.NewRequest(http.MethodGet, address, nil)
		if err != nil {
			h.Config.Logger.Error().Err(err).Msg("failed to create http request")
			return nil
		}
		req = req.WithContext(ctx)

		resp, err := h.Config.HTTPClient.Do(req)
		if err != nil {
			h.Config.Logger.Error().Err(err).Msgf("failed to check the health of node '%d'", node.ID)
			return nil
		}
		defer resp.Body.Close() // nolint: errcheck

		if resp.StatusCode == http.StatusOK {
			return nil
		}
		h.Config.Logger.Error().Msgf(
			"invalid status code '%d' from node '%d'", resp.StatusCode, node.ID,
		)
		return nil
	}
}
