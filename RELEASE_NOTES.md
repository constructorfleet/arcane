# 3.0.1-cf.112

- Use Node for resource-list scope filters and optional scope columns across the shared table, including Swarm resources. Use All Nodes in resource destination/prune selectors and scoped-list return actions.
- Align image pull/push targets, container update targets, active volume mount records, multi-node coverage warnings, resource-scope errors and resource-list appearance settings with the Node terminology contract.
- Preserve Agent terminology for registration, connection/software administration and the Agents pages. Resource IDs, permissions, filter keys and request routing are unchanged.
- Validation: full formatting/lint, all 25 frontend unit tests and the production build passed locally. Required Gitea checks run on the final PR head; authenticated live acceptance follows publication. This wording-only batch does not implement the remaining action-menu, relationship, header or Overview changes.
