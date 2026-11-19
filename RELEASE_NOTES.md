# 3.0.1-cf.9

## Changes

- Add View container actions to global, node, stack, and service task views, including mobile task cards (#8).
- Resolve the task's Docker node to a verified connected agent, inspect the exact container there, and verify its Swarm task label before navigating with an explicit agent URL.
- Show unavailable states for unassigned, removed, inaccessible, or mismatched containers. Historical tasks can open a retained container; no querying-manager fallback is used.

## Validation

- Eight target-policy tests pass for independent agents, mismatches, ambiguous/offline agents, missing containers, and retired local environment IDs. Frontend unit tests now run in required CI alongside build/type/lint checks.
- Required release-contract, full Go tests, and frontend checks run before squash merge. Live navigation and broader console/exec behavior require separate acceptance.
