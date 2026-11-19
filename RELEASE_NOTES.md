# 3.0.1-cf.20

## Fixes

- Use the same verified per-node observations for network details and removal eligibility (#12). A known remote container disables Remove even when the selected node has no containers.
- Keep Remove disabled while requested observations are pending, denied or unavailable, retaining per-agent errors and Retry. Observations still do not establish complete cluster-wide absence; service guards and Docker remain removal authorities.
- Keep late container-sort responses tied to the viewed network, query agent and requested sort.

## Validation and limits

- Reproduced the remaining detail-page gap on cf.15 without executing deletion. The grouped row already disabled Delete correctly.
- Validate existing network business coverage, frontend tests/build, format/lint and required CI before merge. Runtime empty-node/remote-consumer, refresh and retry acceptance follows publication; no production network deletion is used as QA.
