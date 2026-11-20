# 3.0.1-cf.69

- Remove saved image-update checks for the exact deleted project in the existing project-metadata transaction (#167). Other project, global image, container and reference checks remain intact.
- The shared deletion helper also covers existing filesystem reconciliation and failed-create rollback callers. No new lifecycle service, schema migration or legacy-orphan purge is introduced.
- Reconcile the checklist with verified cf.67 revision retention and cf.68 unchanged-source Save acceptance (#147/#144); full project-route QA remains #150.

Validation: extended existing project-destruction coverage reproduces two orphaned check rows on canonical cf.68. The correction removes those rows, preserves four unrelated scopes, rolls project/check deletion back on transaction failure, and retains checks/files/project state after failed Compose down. Focused owning checks pass; full format/lint, owning-package checks and release validation are recorded in the PR.

No live project was created to reproduce this finding, and no production database cleanup was performed. Release/runtime and live owned-project lifecycle acceptance remain pending. Existing orphaned records are not purged; the rollback check covers database metadata.
