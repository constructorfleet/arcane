# 3.0.1-cf.63

- Discover registry namespaces/projects, then repositories and tags without entering every repository manually (#146 / #40). Standard Docker/OCI catalog browsing remains supported; when it is unavailable, an available Harbor project/repository API uses the existing saved credential.
- Browse a specific project even when global project listing is denied, while retaining a configured URL namespace boundary. Known repositories remain an optional fallback for registries without discovery support. Add/Help guidance distinguishes host-only discovery, optional project scope and repository shortcuts.
- Paginate namespaces and repositories; bound provider requests/responses and reject incomplete or repeated pages. Upstream authentication failures retain their category, credentials are not forwarded across redirects, and existing TLS/authentication settings remain unchanged.

Validation: fake-HTTP registry business tests (including CLI-style and robot credentials, project-list denial, pagination, shared request budgets, malformed/oversized/incomplete responses and redirects); eight actual client-component DOM cases and nine owning-loader cases; ten frontend tests, production build, full format/lint and release tooling checks. Child UI/API boundaries are stubbed in isolated frontend checks.

Live private-Harbor browsing with the user's existing connection is pending release/runtime acceptance. Project discovery remains subject to the registry account's permissions; no access is granted or credentials changed. No live registry writes, pulls or pushes were performed for this implementation.
