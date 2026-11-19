# 3.0.1-cf.12

## Fixes

- Show the existing Swarm nodes table on the Cluster page, including per-node status, availability, and permission-gated node actions (#9).
- Add the missing Pause availability action alongside Drain and Active in the shared node menu. Requests retain the exact node ID and query-agent scope.
- Label cluster inspection as cluster-wide administration, separate from node availability.

## Validation and limits

- Validate frontend build/type/lint, existing frontend tests, and required CI before merge.
- Live QA must verify the node list and action scope. Production nodes are not drained or demoted as a UI smoke test.
