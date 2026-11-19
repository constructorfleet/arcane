# 3.0.1-cf.24

## Native Swarm service inspection

- Preserve Docker SDK service specifications using their native JSON encoding before including them in Arcane responses. Empty Global and GlobalJob mode markers and explicit zero replica/completion values now survive inspection instead of disappearing under JSON v2 omission rules. Refs #66.
- Keep the existing mode decoder and native Docker create/update/scale behavior; no frontend mode guessing or global JSON serializer change is introduced.

## Validation

- Existing service-mode business coverage now exercises inspection through response serialization for replicated, zero-replica, replicated-job, global and global-job modes, alongside the existing scale rules.
- Swarm/authz and API checks, existing frontend tests, production build, formatting and full lint pass locally.
- Runtime mode display acceptance follows normal publication and rollout. No production service is mutated for QA.
