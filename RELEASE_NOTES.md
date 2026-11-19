# 3.0.1-cf.8

## Fixes

- Resolve a node's independently enrolled global agent from its reported Docker node identity without requiring a stored parent-agent binding (#5).
- Show each node agent's name, UUID, and connection state in the Nodes table. Label the cluster query source separately as Query Agent.
- Reading the Nodes page no longer tries to rebind all agents to every manager. Discovered agents remain separate from explicitly attached agents in the dialog.
- Preserve explicit bindings and ambiguous-agent detection; multiple agents claiming one node are not silently selected.

## Validation

- Updated existing node-coverage tests exercise manager/worker roles with explicit bindings, independent enrollment, and ambiguous matches.
- Required release-contract, Go, and frontend checks run before merge. Live multi-node UI acceptance remains separate from automated checks.
