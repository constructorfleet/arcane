# 3.0.1-cf.27

## Persistent agent build workspaces

- Add a dedicated node-local build volume mounted at `/builds` to the standalone and global Swarm agent examples. Set the canonical `ARCANE_BUILDS_DIRECTORY` explicitly. Refs #71.
- Keep build files separate from enrolled identity state. Preserve global Linux-node coverage, enrollment secrets and stop-first updates.

## Validation

- Both examples parse successfully using Docker Compose / Docker stack config with placeholder inputs; no example stack was deployed. Formatting, full lint and release contract pass locally.
- The equivalent homelab deployment change (homelab-ansible PR #11) is already deployed: all five agents have the writable local volume. UI file creation/build passed, and both owned files retained exact hashes through cf.24 → cf.25 replacement and reopened without saving. Exact fixture-file cleanup completed afterward.

## Limits

- These examples require the patched agent image and reachable control plane. No identity, shared-storage policy, RBD attachment or data-directory overlap validation changed. Build workspace files remain local to their selected node.
