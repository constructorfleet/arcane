# 3.0.1-cf.64

## Fixed

- Registry project discovery now uses the same HTTPS-first negotiation as OCI browsing. Allowing insecure connections permits a plain HTTP fallback instead of forcing project API requests onto HTTP.
- Discovery probes reject redirects and foreign origins. Authenticated project requests remain on the configured host, do not follow redirects, and retain certificate verification. Probe attempts reserve slots in the existing shared request budget.

## Validation and limits

- Existing browse tests cover verified HTTPS with insecure fallback enabled, explicitly permitted HTTP, refused HTTP fallback, HTTPS permission errors, redirect boundaries, and bounded project enumeration. The trusted TLS case reproduces the previous forced-HTTP failure against canonical cf.63.
- This fixes the cf.63 live transport mismatch reported in #160 (parent #146/#40). Existing-profile namespace/repository/tag UI acceptance remains pending the cf.64 rollout; no registry credentials, permissions, TLS flags, pulls, or pushes were changed by this work.
