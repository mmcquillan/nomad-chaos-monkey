package fault

import (
	"context"

	nomad "github.com/hashicorp/nomad/api"
)

// StopJobFault deregisters a job entirely, testing service-discovery teardown
// and re-registration when the job is eventually re-submitted.
type StopJobFault struct{}

func (f *StopJobFault) Name() string     { return "stop-job" }
func (f *StopJobFault) Kind() TargetKind { return JobTarget }

func (f *StopJobFault) Apply(ctx context.Context, client *nomad.Client, t Target) error {
	_, _, err := client.Jobs().Deregister(t.Job.ID, false, nil)
	return err
}
