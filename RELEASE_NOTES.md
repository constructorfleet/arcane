# 3.0.1-cf.40

- Keep Docker-dependent vulnerability scan and risk jobs on node agents and legacy Docker-enabled managers. Control-plane startup reconciles away their old schedules, and settings changes cannot register them again (#97).
- Reject accidental node-local risk computation on the control plane with a classified conflict instead of attempting to use a missing Docker socket. Agent aggregation and failed-environment coverage remain available.

Existing job-subscription and risk-aggregation business cases exercise control-plane and Docker-enabled modes. Required formatting, lint and CI results are recorded in the fixing PR. Release publication and normal deployment are separate from runtime acceptance: verify manager job metadata and agent coverage after rollout. No production scan or job is triggered by the isolated tests. This corrects these vulnerability jobs; other Docker-dependent background jobs and broader authorization acceptance are outside this change.
