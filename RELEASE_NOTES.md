# 3.0.1-cf.14

## Changes

- Group confirmed shared overlay network records by cluster and Docker network ID before filtering, pagination, and counts (#11). Keep independent or unverified identities separate.
- Clamp saved pagination to the remaining grouped results instead of leaving an out-of-range empty page.
- Label the summary count by observed container use instead of claiming networks are unused cluster-wide.
- Preserve reporting agents and node-local usage; show per-agent container/peer observations with exact-agent container links and partial-failure states.
- Verify cluster/network/node identity before accepting each detail observation. Label query-agent and observed-node scope explicitly.
- Exclude synthetic Docker load-balancer/sandbox endpoints from container links and consumer guards; retain actual containers, including helpers.
- Block removal when observed container endpoints or Swarm service references remain, including services with zero replicas. Swarm removal requires a manager agent; failed identity/service checks fail closed (#12).

## Validation and limits

- Validate network business tests, frontend tests/build, full format/lint and required CI before merge. No production network deletion is used for acceptance.
- Workers that cannot report a cluster ID remain separate. Observations are not a complete cluster inventory or a transactional proof of no remote standalone consumers; Docker remains the final removal authority.
