# 3.0.1-cf.6

## Fixes

- Generate source and pull-request links from the public WEB_URL release metadata instead of runner-internal Gitea addresses.
- Keep the internal API transport separate from user-facing web links; preserve source tags and image digest references.

## Validation

- Release tooling covers draft generation with an internal API URL and a public web URL, including an internal PR URL returned by the API.
- cf.5 release-note web links were corrected without changing its source tag or image digests.
- Required release-contract, Go, and frontend checks run before merge. No application behavior or deployment configuration changes.
