# 3.0.1-cf.5

## Fixes

- Fix Harbor links in release notes: link the repository UI and exact artifact metadata instead of a nonexistent artifact UI route.
- Restore explicit per-job Buildx selection so the runner environment cannot silently redirect image builds to a different builder.
- Preserve version tags, source/PR links, and immutable pull references.

## Validation and limitations

- Harbor's deployed UI code identifies the repository artifacts-tab route; its v2 API serves exact digest metadata.
- The former artifact UI URL was browser-tested and returned 404. The corresponding metadata endpoint returns the expected digest.
- Required release-contract, Go, and frontend checks run before merge. Application behavior and deployment configuration are unchanged.
