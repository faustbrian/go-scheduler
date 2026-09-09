# Documentation

Start with [installation](../README.md#installation), the
[five-minute quickstart](../README.md#five-minute-quickstart), and
[scheduler-versus-sequencer selection](../README.md#scheduler-or-sequencer).
The [package and adapter map](../README.md#package-and-adapter-selection)
identifies each optional boundary and its owner.

## Build and integrate

- [Examples](../examples/README.md)
- [API reference](api.md)
- [Service lifecycle integration](service-integration.md)
- [Composition recipes](composition-recipes.md)
- [Lease backends and fencing](leases.md)
- [Missed runs, time zones, and DST](time-and-missed-runs.md)
- [Queue dispatch and idempotency](dispatch-and-idempotency.md)

## Deploy and operate

- [Kubernetes architecture](kubernetes.md)
- [Operations and recovery](operations.md)
- [Security](security.md)
- [Troubleshooting](troubleshooting.md)
- [Performance](performance.md)
- [Benchmark baseline](benchmark-baseline.md)
- [Resilience and resource budgets](resilience.md)
- [Threat model](threat-model.md)

## Adopt and maintain

- [Laravel migration](laravel-migration.md)
- [Compatibility](compatibility.md)
- [FAQ](faq.md)
- [Changelog](../CHANGELOG.md)
- [Contributing](../CONTRIBUTING.md)
- [Support](../SUPPORT.md)
- [Security reporting](../SECURITY.md)
- [License](../LICENSE)
- [Versioned Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)

The scheduler coordinates decisions. It is not a workflow engine, queue,
worker runtime, Kubernetes controller, or exactly-once system.
