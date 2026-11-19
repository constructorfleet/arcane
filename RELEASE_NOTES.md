# 3.0.1-cf.18

## Fixes

- Test registry endpoint reachability and Basic/Bearer authentication directly over registry HTTP, without requiring a manager Docker socket (#40).
- Contact anonymous endpoints instead of reporting success without a request. Bound checks by the registry timeout and preserve cancellation and TLS verification.
- Distinguish a challenged credential check from unverified credentials on public endpoints; repository permissions are not claimed by a connection test.
- Show catalog errors prominently without describing denied access as a proven unsupported catalog. Preserve known-repository browsing (#50).

- Add a registry-tag Pull action that reuses the existing pull sheet and agent endpoint. Show the selected agent and pin that destination through submission; disconnected or unauthorized targets cannot silently fall back to another node.

## Validation and limits

- The Docker-socket failure and catalog permission error were independently reproduced on cf.12; opening a known Harbor repository already worked via HTTP.
- Validate registry business tests, frontend build/tests, full format/lint and required CI before merge. Pull/build/push execution remains agent-scoped; this does not complete every registry feature in #40.
