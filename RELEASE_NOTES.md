# 3.0.1-cf.28

## Template light and dark icons

- Retain `x-arcane.icon-light` and `icon-dark` when parsing saved templates and expose both resolved URLs to the gallery and template picker. Reuse the existing icon catalog, viewing-user preference and theme selection. Refs #73 / #17.
- Derive variants from saved Compose on catalog reads so previously saved variant-only templates work without editing/resaving or schema migration. Keep canonical source/identifiers unchanged; local catalog responses continue omitting Compose and dotenv contents.
- Preserve generic icon URLs, catalog identifiers and remote-registry icon fallbacks. Light artwork is used for a dark theme and dark artwork for a light theme, matching existing project/container behavior.

## Validation

- Existing template service coverage exercises preexisting variant-only content with dotenv interpolation, catalog preference/default resolution, generic/explicit icons, list-content omission and unchanged database/cache identifiers. Existing template parser tests remain passing.
- Backend template/project tests, frontend tests (8), production frontend build, format/full lint and release contract pass locally. Runtime light/dark and no-resave acceptance follows normal publication/rollout; the owned never-deploy template retains its cf.25 failing variant-only baseline.

## Limits

- Remote catalogs use their registry icon until Compose is downloaded; listing does not fetch every remote template. General template deployment, extension application, custom bang tags and final fork branding remain separate #17 acceptance/design work.
