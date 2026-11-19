# 3.0.1-cf.16

## Build stability

- Bound manager and agent image compilation with `GOMAXPROCS=4` and `go build -p 2`, matching the existing pull-request Go validation limits.
- Keep this limit within the build-stage command. Runtime container configuration continues through the existing deployment configuration.

## Validation and limits

- Severe recent guest memory/CPU/I/O pressure was observed on swarm05 during the cf.15 publication/rollout window; its exact initiating process is unproven (#42). This limits our compilation concurrency as a mitigation, not a claim that the incident's root cause is resolved.
- Validate release tooling, frontend build, full format/lint and required CI before merge. Published image builds and normal rollout must still pass verification.
