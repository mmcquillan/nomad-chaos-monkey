package fault

import (
	"context"
	"time"

	nomad "github.com/hashicorp/nomad/api"
)

// DrainNodeFault initiates a node drain, forcing all allocations to be
// rescheduled elsewhere. This is the highest-impact node-level fault.
type DrainNodeFault struct {
	Deadline time.Duration
}

func (f *DrainNodeFault) Name() string     { return "drain-node" }
func (f *DrainNodeFault) Kind() TargetKind { return NodeTarget }

func (f *DrainNodeFault) Apply(ctx context.Context, client *nomad.Client, t Target) error {
	spec := &nomad.DrainSpec{
		Deadline:         f.Deadline,
		IgnoreSystemJobs: false,
	}
	_, err := client.Nodes().UpdateDrain(t.Node.ID, spec, false, nil)
	return err
}

// IneligibleNodeFault marks a node as ineligible for scheduling without
// migrating existing allocations, simulating a soft partition.
type IneligibleNodeFault struct{}

func (f *IneligibleNodeFault) Name() string     { return "ineligible-node" }
func (f *IneligibleNodeFault) Kind() TargetKind { return NodeTarget }

func (f *IneligibleNodeFault) Apply(ctx context.Context, client *nomad.Client, t Target) error {
	_, err := client.Nodes().ToggleEligibility(t.Node.ID, false, nil)
	return err
}
