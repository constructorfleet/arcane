# 3.0.1-cf.65

- Show one logical entry for a Docker-global plugin volume with the same name and driver in the same observed Swarm (#13 / #14). Preserve the individual agent records under that entry. Local volumes, different clusters/drivers and unknown cluster identities remain separate.
- Select the node with the active application attachment for volume details and Workspace, using fresh container observations. Re-resolve the selected attachment before opening Workspace; retain unsaved edits on their original agent. An inactive or ambiguous attachment shows an explicit state without attempting to mount the volume on another node.
- Group before filtering, sorting, pagination and usage counts. Show partial agent coverage and retain explicit per-node management links. Logical entries exclude bulk/resource deletion while normal authorized workspace file editing remains available.

Validation: existing volume service coverage extended for grouping, identity boundaries, counts and attachment selection; 16 actual owning page-loader cases, 23 workspace owner cases and 23 compiled component SSR cases with service/framework/child boundaries stubbed; production build, full format/lint and release contract/tool checks.

Live single-entry RBD listing and active-node workspace browsing remain pending release/runtime acceptance. Grouping follows Docker global-volume scope within the observed Swarm; it does not infer physical storage identity across clusters. No plugin changes, cross-node mounts or destructive volume operations were performed for this implementation.
