# 3.0.1-cf.68

- Stop container update activity from reading a departed page's missing container during navigation (#150, upstream #4925). Keep the canonical Docker container ID, including when the URL uses a short ID or name.
- Ignore refreshes after the view unmounts or loses its container, and discard delayed replies after its agent or container changes.

Validation: a client DOM check using the pinned SvelteKit root, actual update tracker and the owning callback/teardown reproduces the pre-flush callback failure before the change. The corrected callback handles that transition and preserves same-component navigation. Isolated checks also cover canonical-ID aliases and delayed refresh replies. Full repository checks and release acceptance are recorded in the PR.

The lifecycle check uses a page subset and synthetic navigation parent. Complete router loader/error-boundary behavior and the full live project Files → Updates → Container sequence remain separate acceptance work; this release does not claim the complete upstream report is resolved.
