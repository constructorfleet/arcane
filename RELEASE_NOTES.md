# 3.0.1-cf.43

- Start added user-role assignment rows without a selected role or scope, instead of implicitly choosing the first role (Admin) and Global (#103).
- Require explicit role and scope selection before submitting user creation or updates. Incomplete rows cannot become global assignments, and the form reports how to complete them.
- Preserve existing assignments and deliberately selected global-role quick presets.

Production frontend build, required formatting/lint, existing frontend tests and exact-head CI are recorded in the PR. Post-release UI acceptance will use canceled drafts without creating another account or changing any existing grants. Backend authorization remains authoritative; broader scoped/OIDC acceptance stays open in #63.
