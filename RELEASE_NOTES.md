# 3.0.1-cf.17

## Volume consumer provenance

- Mark known Arcane volume helper containers distinctly in the volume consumer overview, using the existing internal/helper label contract (#13).
- Show unavailable container metadata with the actual API error instead of silently presenting a short ID as complete consumer details.
- Explain that helper mounts still count as real consumers. Volume in-use and attachment policy are unchanged.

## Validation and limits

- Validate frontend tests/build, full format/lint and required CI before merge. Helper presentation requires live acceptance after publication.
- This improves consumer provenance; shared storage identity/grouping and automatic attachment-aware navigation remain open. Current RBD plugin Status exposes size alone and does not provide enough storage identity for safe cross-node grouping.
