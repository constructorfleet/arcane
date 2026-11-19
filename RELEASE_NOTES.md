# 3.0.1-cf.21

## Config content permissions

- Return config metadata without content from list, aggregate and labels-only update responses (#61). Config inspection continues through the separate read-authorized endpoint.
- Apply the metadata projection again when aggregating responses, including older agents during rollout. Preserve resource identity, labels, version, query-agent provenance and pagination.
- Preserve existing config bytes when updating labels. The UI already fetches read-authorized inspection data before opening the editor, so authorized content display continues through that path.

## Validation and limits

- Engine list responses include config Data; the old API forwarded it under list permission, and labels-only updates returned it under update permission. This is source-confirmed; no nonadministrator account was created or live unauthorized access attempted.
- Validate existing config service tests for metadata projection, retained inspection content and unchanged Engine update data; run frontend build/tests, full format/lint, release contract and required CI before merge.
- Runtime acceptance will use an owned synthetic config. No production config content or consumers are changed for QA.
- Standalone image Push PR #60 is held and will advance to cf.22 after this fix.
