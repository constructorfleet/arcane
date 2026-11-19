# 3.0.1-cf.25

## Template catalog icons

- Resolve saved and remote template icon identifiers through the existing icon catalog when producing API responses, honoring the viewing user's catalog preference. Explicit HTTP(S) icon URLs are preserved. Refs #67.
- Apply the same presentation to paginated/all lists, detail/content and mutation responses. Existing stored icon identifiers benefit without resaving; canonical database metadata and shared remote-cache entries remain unchanged.

## Validation

- Existing template-service coverage verifies local/remote identifiers, explicit URLs, preferred/default catalogs and unchanged database/cache metadata alongside type filtering. Template-service/API tests, frontend tests/build, formatting, full lint and release contract pass locally.
- Runtime acceptance follows normal publication and rollout. The cf.22 owned template round trip established the catalog fallback and successful explicit interpolated URL rendering.

## Limits

- Separate light/dark template metadata and deployment/branding acceptance remain tracked in #17. This fix does not deploy templates or rename their syntax.
