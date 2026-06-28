package selector

import (
	"context"
	"errors"
	"math/rand"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/config"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/fault"
)

var ErrNoTargets = errors.New("no eligible targets found")

type Selector struct {
	client *nomad.Client
	cfg    *config.Config
}

func New(client *nomad.Client, cfg *config.Config) *Selector {
	return &Selector{client: client, cfg: cfg}
}

func (s *Selector) PickFor(ctx context.Context, kind fault.TargetKind) (fault.Target, error) {
	switch kind {
	case fault.AllocationTarget:
		return s.pickAlloc(ctx)
	case fault.NodeTarget:
		return s.pickNode(ctx)
	case fault.JobTarget:
		return s.pickJob(ctx)
	default:
		return fault.Target{}, errors.New("unknown target kind")
	}
}

func (s *Selector) pickAlloc(ctx context.Context) (fault.Target, error) {
	// Build allowed job set first to avoid per-alloc API calls for meta checks.
	allowed, err := s.allowedJobIDs()
	if err != nil {
		return fault.Target{}, err
	}

	allocs, _, err := s.client.Allocations().List(&nomad.QueryOptions{
		Namespace: s.cfg.Namespace,
	})
	if err != nil {
		return fault.Target{}, err
	}

	var candidates []*nomad.AllocationListStub
	for _, a := range allocs {
		if a.ClientStatus != "running" || a.DesiredStatus != "run" {
			continue
		}
		if _, ok := allowed[a.JobID]; !ok {
			continue
		}
		candidates = append(candidates, a)
	}

	if len(candidates) == 0 {
		return fault.Target{}, ErrNoTargets
	}
	return fault.Target{Alloc: candidates[rand.Intn(len(candidates))]}, nil
}

func (s *Selector) pickNode(ctx context.Context) (fault.Target, error) {
	nodes, _, err := s.client.Nodes().List(&nomad.QueryOptions{})
	if err != nil {
		return fault.Target{}, err
	}

	var candidates []*nomad.NodeListStub
	for _, n := range nodes {
		if n.Status != "ready" || n.SchedulingEligibility != "eligible" || n.Drain {
			continue
		}
		if s.cfg.NodeClass != "" && n.NodeClass != s.cfg.NodeClass {
			continue
		}
		if s.cfg.Datacenter != "" && n.Datacenter != s.cfg.Datacenter {
			continue
		}
		candidates = append(candidates, n)
	}

	if len(candidates) == 0 {
		return fault.Target{}, ErrNoTargets
	}
	return fault.Target{Node: candidates[rand.Intn(len(candidates))]}, nil
}

func (s *Selector) pickJob(ctx context.Context) (fault.Target, error) {
	jobs, _, err := s.client.Jobs().List(&nomad.QueryOptions{
		Namespace: s.cfg.Namespace,
	})
	if err != nil {
		return fault.Target{}, err
	}

	var candidates []*nomad.JobListStub
	for _, j := range jobs {
		if j.Status != "running" {
			continue
		}
		if !s.jobTypeAllowed(j.Type) {
			continue
		}
		if !s.jobIDAllowed(j.ID) {
			continue
		}
		if !s.metaAllowed(j) {
			continue
		}
		candidates = append(candidates, j)
	}

	if len(candidates) == 0 {
		return fault.Target{}, ErrNoTargets
	}
	return fault.Target{Job: candidates[rand.Intn(len(candidates))]}, nil
}

// allowedJobIDs returns a set of job IDs that pass all job-level filters.
func (s *Selector) allowedJobIDs() (map[string]struct{}, error) {
	jobs, _, err := s.client.Jobs().List(&nomad.QueryOptions{
		Namespace: s.cfg.Namespace,
	})
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]struct{}, len(jobs))
	for _, j := range jobs {
		if j.Status != "running" {
			continue
		}
		if !s.jobTypeAllowed(j.Type) {
			continue
		}
		if !s.jobIDAllowed(j.ID) {
			continue
		}
		if !s.metaAllowed(j) {
			continue
		}
		allowed[j.ID] = struct{}{}
	}
	return allowed, nil
}

func (s *Selector) jobTypeAllowed(jobType string) bool {
	for _, t := range s.cfg.ExcludeJobTypes {
		if t == jobType {
			return false
		}
	}
	return true
}

func (s *Selector) jobIDAllowed(jobID string) bool {
	if len(s.cfg.JobIDs) == 0 {
		return true
	}
	for _, id := range s.cfg.JobIDs {
		if id == jobID {
			return true
		}
	}
	return false
}

func (s *Selector) metaAllowed(j *nomad.JobListStub) bool {
	if s.cfg.MetaKey == "" {
		return true
	}
	if j.Meta == nil {
		return false
	}
	val, ok := j.Meta[s.cfg.MetaKey]
	return ok && (s.cfg.MetaValue == "" || val == s.cfg.MetaValue)
}
