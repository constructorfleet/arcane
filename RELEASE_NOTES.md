# 3.0.1-cf.48

- Validate user passwords without trimming meaningful boundary whitespace, matching the raw value submitted and the server password policy (#115).
- Preserve username/profile normalization, optional blank passwords on Edit and server composition enforcement. Reuse the explicit form-field preservation option from cf.47.

Validated using synthetic compiled-form cases, existing native password-policy coverage and frontend checks. No account, password, role or credential is changed by this validation; live credential entry is not part of acceptance.
