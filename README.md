# nomad-chaos-monkey

Chaos engineering for Nomad clusters. Periodically injects faults into running jobs, allocations, and nodes to surface reliability issues before they surface in production.

## Usage

```
nomad-chaos-monkey [flags]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--nomad-addr` | `http://127.0.0.1:4646` | Nomad API address |
| `--nomad-token` | | Nomad ACL token |
| `--namespace` | `default` | Nomad namespace to target |
| `--job` | | Specific job IDs to target (default: all eligible) |
| `--node-class` | | Restrict node faults to this node class |
| `--datacenter` | | Restrict to this datacenter |
| `--meta-key` | `chaos.enabled` | Job meta key required for opt-in (empty to disable check) |
| `--meta-value` | `true` | Required value for meta key |
| `--min-healthy` | `2` | Minimum running allocs a job must have before it can be targeted |
| `--exclude-type` | `system,sysbatch` | Job types to exclude |
| `--skip-deploying` | `true` | Skip jobs with an active deployment |
| `--interval` | `5m` | Time between fault injections |
| `--jitter` | `30s` | Max random jitter added to each interval |
| `--blackout` | | Blackout windows in `HH:MM-HH:MM` format (local time, repeatable) |
| `--fault` | `stop-alloc` | Fault types to enable (repeatable) |
| `--dry-run` | `false` | Log what would happen without taking action |

## Fault types

| Fault | Target | Description |
|---|---|---|
| `stop-alloc` | allocation | Stops a running allocation; Nomad reschedules it per the job's reschedule policy |
| `restart-alloc` | allocation | Restarts an allocation in-place without rescheduling |
| `signal-alloc` | allocation | Sends `SIGTERM` to all tasks in an allocation |
| `stop-job` | job | Deregisters a job, testing service-discovery teardown and re-registration |
| `drain-node` | node | Drains all allocations off a node with a 1-hour deadline |
| `ineligible-node` | node | Marks a node ineligible for scheduling without migrating existing allocations |

## Opt-in targeting

By default, only jobs with `meta { chaos.enabled = "true" }` in their jobspec are eligible. To target all jobs regardless of metadata, set `--meta-key ""`.

## Examples

Dry run against a local Nomad dev agent:

```sh
nomad-chaos-monkey --dry-run
```

Target only `stop-alloc` and `drain-node` faults against jobs in the `prod` namespace, every 10 minutes:

```sh
nomad-chaos-monkey \
  --nomad-addr http://nomad.example.com:4646 \
  --nomad-token $NOMAD_TOKEN \
  --namespace prod \
  --fault stop-alloc \
  --fault drain-node \
  --interval 10m \
  --blackout 22:00-06:00
```

Target a specific job with no meta-key requirement:

```sh
nomad-chaos-monkey --job my-service --meta-key "" --min-healthy 3
```

## Building

```sh
go build .
```
