# 3.0.1-cf.29

## Verified feature acceptance

- Reconcile FEATURES.md with bounded live acceptance: fresh OIDC sign-in, native config/secret creation, Compose stack and standalone authoring, fractional CPU create/edit, selected-agent image Push/build/fresh-layer Pull, persistent local build workspaces, and template icon presentation. Link the remaining scope rather than treating partial evidence as full completion.
- Add REG-07 and OBS-03 for selected-agent workspace builds and six-target private Prometheus collection. Homelab PR #12 deployed existing metrics through the monitoring overlay; repeated queries and native Docker container-ID coverage passed on all five nodes.
- Record source/test audits of reusable encrypted inputs and telemetry, alongside the distinction between global environment materialization and native Swarm secrets. Preserve open scoped-authorization, template deployment/theme, registry/cache, RBD and later design work.

## Validation and runtime evidence

- Documentation formatting, full lint and release contract pass locally. This release records behavior verified across cf.20–cf.28 and the homelab configuration rollout; it introduces no new application implementation.
- Current cf.28 manager on swarm03 passes Docker health and database readiness. Five agents are Running, UI Online and metrics-up with unchanged UUIDs; agent images do not define Docker HEALTHCHECKs.

## Limits

- Metrics proof covers six unique task targets, observed series and native ID coverage, warm requests and successful replacement scrapes. Fully isolated cold-cache completeness and broader operational/reconciliation metrics and dashboards remain #32.
- Node05 experienced a global OOM/pressure incident during configuration rollout and recovered without interventions. Collector cardinality tracking has a proven 8 GiB allocation; its policy choice is tracked in #76, separate from RBD handoff #42. The initiating global-OOM allocation remains unproven.
- Native config/secret and template/registry acceptance remain bounded to the recorded fixtures and actors; scoped denial #63 and the actual inactive RBD target #14 still need their missing inputs.
