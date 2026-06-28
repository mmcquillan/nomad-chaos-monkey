package engine

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/config"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/fault"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/safety"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/selector"
)

type Engine struct {
	client   *nomad.Client
	cfg      *config.Config
	registry *fault.Registry
	selector *selector.Selector
	safety   *safety.Checker
}

func New(client *nomad.Client, cfg *config.Config, registry *fault.Registry, sel *selector.Selector, checker *safety.Checker) *Engine {
	return &Engine{
		client:   client,
		cfg:      cfg,
		registry: registry,
		selector: sel,
		safety:   checker,
	}
}

func (e *Engine) Run(ctx context.Context) error {
	if e.registry.Empty() {
		return fmt.Errorf("no faults configured")
	}

	for {
		wait := e.cfg.Interval + jitter(e.cfg.Jitter)
		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}

		if e.inBlackout() {
			slog.Info("in blackout window, skipping")
			continue
		}

		f := e.registry.Pick()

		target, err := e.selector.PickFor(ctx, f.Kind())
		if err != nil {
			slog.Warn("no target available", "fault", f.Name(), "err", err)
			continue
		}

		if err := e.safety.Check(ctx, target); err != nil {
			slog.Warn("safety check failed, skipping", "fault", f.Name(), "err", err)
			continue
		}

		attrs := targetAttrs(target)
		slog.Info("firing fault", append([]any{"fault", f.Name(), "dry_run", e.cfg.DryRun}, attrs...)...)

		if !e.cfg.DryRun {
			if err := f.Apply(ctx, e.client, target); err != nil {
				slog.Error("fault apply failed", "fault", f.Name(), "err", err)
			}
		}
	}
}

func jitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(max)))
}

func (e *Engine) inBlackout() bool {
	if len(e.cfg.Blackouts) == 0 {
		return false
	}
	now := time.Now()
	for _, w := range e.cfg.Blackouts {
		if withinWindow(now, w) {
			return true
		}
	}
	return false
}

// withinWindow checks whether now falls inside a "HH:MM-HH:MM" window.
func withinWindow(now time.Time, window string) bool {
	parts := strings.SplitN(window, "-", 2)
	if len(parts) != 2 {
		slog.Warn("invalid blackout window, ignoring", "window", window)
		return false
	}
	start, err1 := parseHHMM(now, parts[0])
	end, err2 := parseHHMM(now, parts[1])
	if err1 != nil || err2 != nil {
		slog.Warn("could not parse blackout window, ignoring", "window", window)
		return false
	}
	return !now.Before(start) && now.Before(end)
}

func parseHHMM(ref time.Time, hhmm string) (time.Time, error) {
	parts := strings.SplitN(hhmm, ":", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time %q", hhmm)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day(), h, m, 0, 0, ref.Location()), nil
}

func targetAttrs(t fault.Target) []any {
	if t.Alloc != nil {
		return []any{"alloc_id", t.Alloc.ID, "job_id", t.Alloc.JobID, "node_id", t.Alloc.NodeID}
	}
	if t.Node != nil {
		return []any{"node_id", t.Node.ID, "node_name", t.Node.Name}
	}
	if t.Job != nil {
		return []any{"job_id", t.Job.ID}
	}
	return nil
}
