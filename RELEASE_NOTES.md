# 3.0.1-cf.70

- Keep a mounted project's cache observer scoped to the query keys captured when it mounts (#150 / CMP-10). A cache update during navigation no longer reads disappearing route data before the observer unsubscribes.
- Preserve existing project/environment editor resets, cache filtering and delayed-work cancellation. No layout, router, dependency or API changes are required.
- Record cf.69's verified project-check cleanup: ordinary Check→Destroy removed the owned check record, project, source, container and bridge (#167).

Validation: actual pinned SvelteKit Root/Props/RenderNode and project layout, with the exact mounted project cache callback and an isolated cache emitter, reproduce the live undefined.data error before the fix. Captured keys avoid the error under both transition orders and preserve project/environment remounts and query scopes. Full format/lint, production frontend build and required CI are recorded in the PR.

The live cf.69 client transition failed; a cold reload of the same URL succeeded. The owned fixture was fully cleaned and the final core/runtime gate passed. This release's live navigation acceptance remains pending. The reproduction uses a page subset and synthetic cache events, not the complete router or an auto-update installation. The separate debug webhook redaction finding remains #169.
