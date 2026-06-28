package fault

import (
	"context"
	"fmt"

	nomad "github.com/hashicorp/nomad/api"
)

func fetchAlloc(client *nomad.Client, id string) (*nomad.Allocation, error) {
	alloc, _, err := client.Allocations().Info(id, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch alloc %s: %w", id, err)
	}
	return alloc, nil
}

// StopAllocFault stops an allocation; Nomad reschedules it per the job's
// reschedule policy, making this the closest analogue to a process crash.
type StopAllocFault struct{}

func (f *StopAllocFault) Name() string     { return "stop-alloc" }
func (f *StopAllocFault) Kind() TargetKind { return AllocationTarget }

func (f *StopAllocFault) Apply(ctx context.Context, client *nomad.Client, t Target) error {
	alloc, err := fetchAlloc(client, t.Alloc.ID)
	if err != nil {
		return err
	}
	_, err = client.Allocations().Stop(alloc, nil)
	return err
}

// RestartAllocFault restarts the allocation in-place without rescheduling.
// Useful for testing restart hooks and fast-restart behavior.
type RestartAllocFault struct{}

func (f *RestartAllocFault) Name() string     { return "restart-alloc" }
func (f *RestartAllocFault) Kind() TargetKind { return AllocationTarget }

func (f *RestartAllocFault) Apply(ctx context.Context, client *nomad.Client, t Target) error {
	alloc, err := fetchAlloc(client, t.Alloc.ID)
	if err != nil {
		return err
	}
	return client.Allocations().Restart(alloc, "", nil)
}

// SignalAllocFault sends a signal to all tasks in an allocation.
type SignalAllocFault struct {
	Signal string
	Task   string // empty = all tasks
}

func (f *SignalAllocFault) Name() string     { return "signal-alloc" }
func (f *SignalAllocFault) Kind() TargetKind { return AllocationTarget }

func (f *SignalAllocFault) Apply(ctx context.Context, client *nomad.Client, t Target) error {
	alloc, err := fetchAlloc(client, t.Alloc.ID)
	if err != nil {
		return err
	}
	return client.Allocations().Signal(alloc, nil, f.Task, f.Signal)
}
