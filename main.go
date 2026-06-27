package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/config"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/engine"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/fault"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/safety"
	"github.com/mmcquillan/nomad-chaos-monkey/internal/selector"
	"github.com/spf13/cobra"
)

func main() {
	cfg := &config.Config{}

	root := &cobra.Command{
		Use:   "nomad-chaos-monkey",
		Short: "Chaos engineering for Nomad clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg)
		},
	}

	f := root.Flags()
	f.StringVar(&cfg.NomadAddr, "nomad-addr", "http://127.0.0.1:4646", "Nomad API address")
	f.StringVar(&cfg.NomadToken, "nomad-token", "", "Nomad ACL token")
	f.StringVar(&cfg.Namespace, "namespace", "default", "Nomad namespace to target")
	f.StringSliceVar(&cfg.JobIDs, "job", nil, "Specific job IDs to target (default: all eligible)")
	f.StringVar(&cfg.NodeClass, "node-class", "", "Restrict node faults to this node class")
	f.StringVar(&cfg.Datacenter, "datacenter", "", "Restrict to this datacenter")
	f.StringVar(&cfg.MetaKey, "meta-key", "chaos.enabled", "Job meta key required for opt-in (empty to disable check)")
	f.StringVar(&cfg.MetaValue, "meta-value", "true", "Required value for meta key")
	f.IntVar(&cfg.MinHealthyAllocs, "min-healthy", 2, "Minimum running allocs a job must have before it can be targeted")
	f.StringSliceVar(&cfg.ExcludeJobTypes, "exclude-type", []string{"system", "sysbatch"}, "Job types to exclude")
	f.BoolVar(&cfg.SkipIfDeploying, "skip-deploying", true, "Skip jobs with an active deployment")
	f.DurationVar(&cfg.Interval, "interval", 5*time.Minute, "Time between fault injections")
	f.DurationVar(&cfg.Jitter, "jitter", 30*time.Second, "Max random jitter added to each interval")
	f.StringSliceVar(&cfg.Blackouts, "blackout", nil, "Blackout windows in HH:MM-HH:MM format (local time)")
	f.StringSliceVar(&cfg.Faults, "fault", []string{"stop-alloc"}, "Fault types to enable")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "Log what would happen without taking action")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	nomadCfg := nomad.DefaultConfig()
	nomadCfg.Address = cfg.NomadAddr
	nomadCfg.SecretID = cfg.NomadToken

	client, err := nomad.NewClient(nomadCfg)
	if err != nil {
		return fmt.Errorf("nomad client: %w", err)
	}

	registry := fault.NewRegistry(cfg.Faults)
	sel := selector.New(client, cfg)
	checker := safety.New(client, cfg)
	eng := engine.New(client, cfg, registry, sel, checker)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("nomad-chaos-monkey starting", "dry_run", cfg.DryRun, "faults", cfg.Faults, "interval", cfg.Interval)
	return eng.Run(ctx)
}
