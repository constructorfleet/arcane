# 3.0.1-cf.67

- Retain managed config and secret revisions referenced by any current service, rollback PreviousSpec or retained task, including runtime credential-spec configs (#147). Desired resources and native reference names/IDs are preserved before automatic post-deploy cleanup.
- Defer all automatic config/secret removal when service or task reference inventory is unavailable. Existing ownership-label selection, native in-use handling and explicit stack teardown remain unchanged.
- Reconcile the volume checklist with cf.65's verified single-entry RBD browsing, active-agent routing, distinct local records, pagination/sorting and attachment audit.

Validation: existing cleanup business coverage extended with the actual pinned Docker SDK and fake HTTP; canonical cf.64/cf.65 cleanup fails three intended retention/inventory-error cases, while the corrected complete Swarm package passes. Full format/lint passed on the source candidate. A fixed read-only native ServiceList query confirmed PreviousSpec metadata and config/secret references are present; no config or secret values were output.

Final-base checks and release/runtime acceptance are recorded in the PR. Live owned A/B/no-op revision acceptance remains pending. This preserves future observed dependencies and does not reconstruct already-deleted resources. The reference snapshot is not atomic against concurrent external Docker changes; historical tasks conservatively retain revisions until Engine history expires. No production config/secret removal, rollback or account change was performed for this implementation.
