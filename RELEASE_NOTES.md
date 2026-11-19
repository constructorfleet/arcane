# 3.0.1-cf.30

## Node role authorization

- Require the target agent’s `swarm-nodes:promote` or `swarm-nodes:demote` permission when the generic node PATCH includes a role. Node-update access alone cannot change a role through this alternate path.
- Reject unknown or empty submitted roles before Docker access. Ordinary label and availability edits retain their existing permission; dedicated promote/demote actions retain theirs.
- Export the role-dependent permission conditions with the PATCH operation. Refs #78, #34 and #63.

## Validation

- Extended existing permission-condition coverage. Relevant authorization, middleware, node and API checks, formatting, full lint and release contract passed.
- Seventeen synthetic requests exercised the actual node routes: denied/wrong-scope requests and invalid roles stopped before the service boundary; correctly authorized role requests and ordinary label/availability requests reached a deliberately stopped synthetic service. No Docker client or live account was used.
- Production node roles and availability are unchanged. The limited live QA account required by #63 is still unavailable; synthetic authorization evidence must not be reported as that live acceptance.
