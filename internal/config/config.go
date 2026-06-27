package config

import (
	"fmt"
	"time"
)

type Config struct {
	NomadAddr  string
	NomadToken string

	// Target selection
	Namespace   string
	JobIDs      []string
	NodeClass   string
	Datacenter  string
	MetaKey     string
	MetaValue   string

	// Safety
	MinHealthyAllocs int
	ExcludeJobTypes  []string
	SkipIfDeploying  bool

	// Scheduling
	Interval  time.Duration
	Jitter    time.Duration
	Blackouts []string // HH:MM-HH:MM

	// Execution
	Faults []string
	DryRun bool
}

func (c *Config) Validate() error {
	if c.NomadAddr == "" {
		return fmt.Errorf("nomad-addr is required")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if c.Jitter < 0 {
		return fmt.Errorf("jitter must be non-negative")
	}
	if c.MinHealthyAllocs < 0 {
		return fmt.Errorf("min-healthy must be non-negative")
	}
	if len(c.Faults) == 0 {
		return fmt.Errorf("at least one --fault must be specified")
	}
	return nil
}
