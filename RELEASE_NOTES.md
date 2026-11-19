# 3.0.1-cf.23

## Fix

- Close Tag Image after a successful tag and image refresh. The completed action now clears its busy state before requesting dialog closure; failures retain the dialog and actual error feedback. Refs #64.

## Validation

- The issue was reproduced in cf.22: an owned unique tag was created on swarm03 and the success toast appeared, but the dialog remained open.
- All 8 existing frontend tests, the production build, formatting and full lint passed locally. Runtime closure acceptance follows normal publication and rollout.

## Limits

- No registry access, image tags, or deployment behavior are changed by this UI fix.
