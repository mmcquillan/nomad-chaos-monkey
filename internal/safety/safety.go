package safety

import (
	"context"
	"fmt"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/config"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/fault"
)

type Checker struct {
	client *nomad.Client
	cfg    *config.Config
}

func New(client *nomad.Client, cfg *config.Config) *Checker {
	return &Checker{client: client, cfg: cfg}
}

func (c *Checker) Check(ctx context.Context, t fault.Target) error {
	if t.Alloc != nil {
		return c.checkAlloc(t.Alloc)
	}
	if t.Node != nil {
		return c.checkNode(t.Node)
	}
	if t.Job != nil {
		return c.checkJob(t.Job)
	}
	return nil
}

func (c *Checker) checkAlloc(alloc *nomad.AllocationListStub) error {
	if err := c.checkMinHealthy(alloc.JobID, alloc.Namespace); err != nil {
		return err
	}
	if c.cfg.SkipIfDeploying {
		return c.checkNotDeploying(alloc.JobID, alloc.Namespace)
	}
	return nil
}

func (c *Checker) checkNode(node *nomad.NodeListStub) error {
	if node.Drain || node.SchedulingEligibility != "eligible" {
		return fmt.Errorf("node %s is already draining or ineligible", node.ID)
	}
	return nil
}

func (c *Checker) checkJob(job *nomad.JobListStub) error {
	if err := c.checkMinHealthy(job.ID, job.Namespace); err != nil {
		return err
	}
	if c.cfg.SkipIfDeploying {
		return c.checkNotDeploying(job.ID, job.Namespace)
	}
	return nil
}

func (c *Checker) checkMinHealthy(jobID, namespace string) error {
	if c.cfg.MinHealthyAllocs <= 0 {
		return nil
	}
	allocs, _, err := c.client.Jobs().Allocations(jobID, false, &nomad.QueryOptions{
		Namespace: namespace,
	})
	if err != nil {
		return fmt.Errorf("list allocs for %s: %w", jobID, err)
	}
	healthy := 0
	for _, a := range allocs {
		if a.ClientStatus == "running" && a.DesiredStatus == "run" {
			healthy++
		}
	}
	if healthy <= c.cfg.MinHealthyAllocs {
		return fmt.Errorf("job %s has %d healthy alloc(s), need more than %d to proceed", jobID, healthy, c.cfg.MinHealthyAllocs)
	}
	return nil
}

func (c *Checker) checkNotDeploying(jobID, namespace string) error {
	deployment, _, err := c.client.Jobs().LatestDeployment(jobID, &nomad.QueryOptions{
		Namespace: namespace,
	})
	if err != nil || deployment == nil {
		return nil
	}
	if deployment.Status == "running" || deployment.Status == "initializing" {
		return fmt.Errorf("job %s has an active deployment (status: %s)", jobID, deployment.Status)
	}
	return nil
}
