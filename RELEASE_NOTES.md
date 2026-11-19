# 3.0.1-cf.15

## Fixes

- Defer table filter translation labels until use, avoiding calls into not-yet-initialized translation exports across production chunk cycles (#52).
- Preserve existing filter values, icons, and labels while allowing network details to initialize independently of previous list navigation.

## Validation and limits

- Reproduced fresh network detail failures on cf.14 for both busy and empty node perspectives, while the agent returned HTTP 200. List navigation, grouped search, per-node observations and the exact consumer link worked.
- Validate frontend tests, production build, format/lint and required CI before merge. Fresh deep links, refresh and both node perspectives require live acceptance after publication. No production network deletion is used as a QA test.
