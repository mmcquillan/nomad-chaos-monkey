package fault

import (
	"context"
	"math/rand"
	"time"

	nomad "github.com/hashicorp/nomad/api"
)

type TargetKind int

const (
	AllocationTarget TargetKind = iota
	NodeTarget
	JobTarget
)

// Target holds the selected victim for a fault. Only one field is non-nil per
// invocation, depending on the fault's Kind().
type Target struct {
	Alloc *nomad.AllocationListStub
	Node  *nomad.NodeListStub
	Job   *nomad.JobListStub
}

type Fault interface {
	Name() string
	Kind() TargetKind
	Apply(ctx context.Context, client *nomad.Client, t Target) error
}

type Registry struct {
	faults []Fault
}

// NewRegistry builds a registry containing only the named fault types.
func NewRegistry(enabled []string) *Registry {
	all := map[string]Fault{
		"stop-alloc":      &StopAllocFault{},
		"restart-alloc":   &RestartAllocFault{},
		"signal-alloc":    &SignalAllocFault{Signal: "SIGTERM"},
		"stop-job":        &StopJobFault{},
		"drain-node":      &DrainNodeFault{Deadline: time.Hour},
		"ineligible-node": &IneligibleNodeFault{},
	}

	r := &Registry{}
	for _, name := range enabled {
		if f, ok := all[name]; ok {
			r.faults = append(r.faults, f)
		}
	}
	return r
}

func (r *Registry) Pick() Fault {
	if len(r.faults) == 0 {
		return nil
	}
	return r.faults[rand.Intn(len(r.faults))]
}

func (r *Registry) Empty() bool {
	return len(r.faults) == 0
}
