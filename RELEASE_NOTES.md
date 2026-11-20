# 3.0.1-cf.66

- Enforce the existing password-change requirement for user-owned personal and scoped API keys across typed API routes, Echo middleware and agent proxy validation (#162 / security verification #148).
- Preserve the existing password-change operation allowlist, normal ready-user keys and ownerless environment/agent authentication. No permissions or credentials are changed by this fix.

Validation: six isolated cases reproduce the canonical cf.63 gap through actual Huma, Echo and proxy validators; patched full auth/bootstrap suites pass with ready, disabled, invalid and environment/agent controls. Full format/lint passed on the source candidate. Final-base checks and release/runtime verification are recorded in the PR.

No live password, API-key, account or grant mutation was performed for this validation. The broader public-advisory verification matrix remains open; this change addresses the focused user-owned-key enforcement gap.
