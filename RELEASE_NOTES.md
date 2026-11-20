# 3.0.1-cf.44

- Use the configured password minimum in user create/edit validation, replacing the hardcoded eight-character check (#107).
- Reuse the existing password-policy requirements for new-user guidance and validation feedback. Align the account-security fallback with the server/policy helper default of twelve characters.
- Preserve empty-password edits and externally managed identity behavior. Server enforcement of the configured length/composition rules remains authoritative.

Production frontend build, required formatting/lint, existing frontend tests and exact-head CI are recorded in the PR. Post-release UI checks use blank-password drafts and cancellation without entering a new credential, creating accounts or changing existing passwords/grants. Broader authorization/OIDC acceptance remains #63.
