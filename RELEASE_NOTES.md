# 3.0.1-cf.22

## Agent-scoped image push

- Add standalone image Push on image details, using the existing image-push permission and agent command tunnel. Pin the owning agent, source image ID and chosen existing repository tag through submission (#40).
- Preserve registry ports in Tag Image defaults and show actual tag errors.
- Require an explicit registry host and tag; use Tag Image first to choose a new destination. Verify that the tag still points to the selected source image before pushing. No automatic retag, pull, registry rewrite or service deployment is performed.
- Reuse registry credentials; propagate matching credential-resolution failures rather than falling through silently. Do not retry a rejected push anonymously.
- Record a push activity with progress, cancellation and actual errors, bounded to ten minutes. Require Docker's final digest confirmation before reporting success.
- Reuse one transfer-stream client for pull and push, rejecting incomplete streams without a terminal confirmation.

## Validation and limits

- Validate image/registry business behavior, agent routing and authorization coverage, frontend tests/build, full format/lint, release contract and required CI before merge.
- Runtime acceptance requires the normal release rollout and a disposable registry destination with suitable push permission. No existing release tag is overwritten for QA.
- This does not complete every provider capability or build workflow in #40. Docker has no atomic inspect-and-push operation: the identity guard detects an already moved local tag but cannot lock out concurrent external Docker clients.
