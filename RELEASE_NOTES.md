# 3.0.1-cf.31

## Preserve native service settings during edits

- Keep zero replicas when saving unrelated service metadata. Explicit replica changes accept zero and reject empty, negative, fractional or unsafe counts before submission.
- Preserve untouched Command/Args arrays exactly, including argument boundaries and empty strings. Label-only edits also retain untouched environment, mount and port settings, including native options the simple form does not expose.
- Refs #80 and #81, reproduced with an owned Docker-CLI-created service through swarm03. Its original native ID survived the edit, but cf.28 changed zero replicas to one and split an untouched argument. The harmless exit-only fixture was restored to zero and its original argument array.

## Validation scope

- Twenty checks executing the existing component submission functions against synthetic native specs passed, along with all eight existing frontend tests and the production frontend build. Formatting, full lint and release contract also passed.
- Live acceptance uses only the owned zero-replica fixture. Production service configuration, RBD volumes and node roles are unchanged. Intentional command editing retains the existing whitespace syntax; this fix preserves untouched arrays without reinterpreting them.
