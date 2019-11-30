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
	Update(ctx context.Context, node service.Node) error
}

// HealthConfigCheckRepository is used to count the checks on the nodes.
type HealthConfigCheckRepository interface {
	Increment(ctx context.Context, id int) (int, error)
	Update(ctx context.Context, id, value int) error
}

// HealthConfig used to setup the health internal state.
type HealthConfig struct {
	// Interval used to check the nodes health.
	Interval time.Duration

	// Used to fetch the nodes.
	Repository HealthConfigRepository

	// Used to count the checks on the nodes.
	CheckRepository HealthConfigCheckRepository

	// Concurrency used to check the nodes.
	Concurrency int

	// Max quantity of failures allowed before disabling a node.
	MaxFailures int

	HTTPClient *http.Client
	Logger     zerolog.Logger
}

// Health is used to track the health of the nodes.
type Health struct {
	Config HealthConfig

	add       chan service.Node
	ctx       context.Context
	ctxCancel func()
	nodes     map[int]service.Node
	wg        sync.WaitGroup
}

// Start the process.
func (h *Health) Start() error {
	h.add = make(chan service.Node)
	h.nodes = make(map[int]service.Node)

	if err := h.updateNodes(); err != nil {
		return fmt.Errorf("failed to update the nodes")
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

func (h *Health) updateNodes() error {
	nodes, err := h.Config.Repository.Select(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %w", err)
	}

	for _, node := range nodes {
		h.nodes[node.ID] = node
	}
	return nil
}

func (h *Health) process() {
	defer h.wg.Done()
	h.ctx, h.ctxCancel = context.WithCancel(context.Background())

	var removeList []int
	for {
		select {
		case node := <-h.add:
			h.Config.Logger.Debug().Int("nodeID", node.ID).Msg("received node creation notification")
			h.nodes[node.ID] = node
			continue
		case <-h.ctx.Done():
			return
		case <-time.After(h.Config.Interval):
		}
		h.Config.Logger.Debug().Msg("New health check cycle")

		g, gctx := errgroup.WithContext(h.ctx)
		ratelimit := make(chan struct{}, h.Config.Concurrency)
		for i, node := range h.nodes {
			ratelimit <- struct{}{}
			var (
				i    = i
				node = node
			)
			g.Go(func() error {
				healthy := h.check(gctx, ratelimit, node)
				constraint, err := h.checkConstraint(gctx, healthy, node)
				if err != nil {
					h.Config.Logger.Error().Err(err).Msg("failed to check the constraints")
					return nil
				}
				if constraint {
					removeList = append(removeList, i)
				}
				return nil
			})
		}
		g.Wait() // nolinter: errcheck

		for _, id := range removeList {
			delete(h.nodes, id)
		}
	}
}

func (h *Health) check(ctx context.Context, rl <-chan struct{}, node service.Node) bool {
	defer func() { <-rl }()

	address := fmt.Sprintf("%s/health", node.Address)
	h.Config.Logger.Debug().Str("endpoint", address).Msg("Executing health check")
	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		h.Config.Logger.Error().Err(err).Msg("failed to create http request")
		return false
	}
	req = req.WithContext(ctx)

	resp, err := h.Config.HTTPClient.Do(req)
	if err != nil {
		h.Config.Logger.Error().Err(err).Msgf("failed to check the health of node '%d'", node.ID)
		return false
	}
	defer resp.Body.Close() // nolint: errcheck

	if resp.StatusCode != http.StatusOK {
		h.Config.Logger.Error().Msgf(
			"invalid status code '%d' from node '%d'", resp.StatusCode, node.ID,
		)
		return false
	}

	return true
}

func (h *Health) checkConstraint(
	ctx context.Context, healty bool, node service.Node,
) (bool, error) {
	if healty {
		if err := h.Config.CheckRepository.Update(ctx, node.ID, 0); err != nil {
			return false, fmt.Errorf("failed to update the check counter: %w", err)
		}
		return false, nil
	}

	value, err := h.Config.CheckRepository.Increment(ctx, node.ID)
	if err != nil {
		return false, fmt.Errorf("failed to increment the check counter: %w", err)
	}

	if value < h.Config.MaxFailures {
		return false, nil
	}

	node.Active = false
	if err := h.Config.Repository.Update(ctx, node); err != nil {
		return false, fmt.Errorf("failed to update the node: %w", err)
	}
	return true, nil
}
