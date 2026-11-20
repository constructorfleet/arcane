# 3.0.1-cf.49

- Clarify English registry overview, empty-state and type guidance for public/private OCI access with optional credentials (#116).
- Explain that Enabled controls automatic image credential matching; explicit connection browsing/testing remains available when disabled. Registry behavior and saved settings are unchanged.

Validated against the owning registry paths and frontend checks. Fresh live guidance acceptance follows deployment; no registry record, credential, TLS flag, image pull or push is changed by this copy update. Broader registry/provider/cache acceptance remains #40/#31.
