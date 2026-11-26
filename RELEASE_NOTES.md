# 3.0.1-cf.105

- Show used and total capacity for the backing filesystem in Volume Workspace and in the all-agent and agent-scoped volume lists. Local directory volumes can share the same filesystem.
- List rows with an existing workspace helper can report usage on the selected agent. Other readable rows can link to Workspace when access is available. Loading the list never mounts volumes.
- Verify filesystem accounting, unavailable readings, agent command routing, and frontend checks. Live RBD usage remains subject to deployment verification.
