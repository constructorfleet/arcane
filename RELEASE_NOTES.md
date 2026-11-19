# 3.0.1-cf.35

- Preserve the complete local path and query when an expired session or unauthenticated page redirects to sign-in (#90). Selected-agent `env`, workspace `tab`, sorting and other query parameters survive the redirect.
- Keep existing authentication-page loop guards, refresh recovery and the root landing-page preference unchanged.
- Extend the existing authentication browser cases to assert the returned path. A network-free harness exercises the actual API failure handler and root-layout guard, including refresh and navigation guards.

Full checks and release verification are recorded in the fixing PR and issue. The browser suite requires its development prerequisites and is not run against the production service. No wrong-node mutation was attempted or observed during QA.
