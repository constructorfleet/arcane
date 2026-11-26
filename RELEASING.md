# Constructorfleet development and releases

Gitea is authoritative: https://gitea.home.prettybaked.com/constructorfleet/arcane.
GitHub is a source mirror; upstream getarcaneapp workflows are not the fork's release pipeline.

## Change workflow

1. Branch from current `main`: `feat/...`, `fix/...`, `docs/...`, `chore/...`, `refactor/...`, `test/...`, `build/...`, `ci/...`, or `perf/...`.
2. Keep each commit coherent: one feature, fix, test update, or documentation change. Do not mix unrelated work or generated files. A PR should have one reviewable purpose.
3. For QA findings, create a [QA bug report](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/new/choose). Include the deployed release, selected agents, reproduction steps, expected behavior, and sanitized evidence. Track one problem per issue and link the relevant FEATURES.md ID.
4. Advance `VERSION` in `RELEASE` from `X.Y.Z-cf.N` to `X.Y.Z-cf.N+1`. A newer upstream base starts at `cf.1`. Every merge, including documentation and CI work, produces a release. Update `RELEASE_NOTES.md` with the exact version, changes, validation, and limitations.
5. Open a Gitea PR with a conventional title such as `fix(volumes): tolerate repeated plugin records`. Link issues with `Fixes #N`, or `Refs #N` for partial work. Keep FEATURES.md aligned with verified behavior.
6. Run relevant checks and report blockers honestly. The required Gitea checks are **Release contract**, **Go tests**, and **Frontend checks**. Main must be current; update the branch and rerun checks after another merge. Seek independent review when available; this solo-maintainer repository does not require an impossible self-approval.
7. Squash-merge the PR through Gitea after checks pass, using its conventional title for the squash commit. Delete the merged branch. Direct pushes, force pushes, merge commits, and rebase merges to main are disabled. Do not override checks.
8. Wait for the release workflow and verify the release and both images before merging the next PR. Observe SwarmCD rollout; do not repeatedly force updates. Mirror the merged main to GitHub, never develop a separate GitHub history.

## Release pipeline

The main push runs `.gitea/workflows/publish-harbor.yml`. It checks immutable tag ownership, builds manager and agent images with provenance and OCI source/version/revision labels, and publishes version and full-commit tags. Only after both builds succeed does it prepare the Gitea release, promote both `latest` tags, and publish the notes. An older run cannot promote latest over a newer main commit. All fork releases currently use SemVer prerelease versions (`-cf.N`) and are marked accordingly in Gitea.

Each `vX.Y.Z-cf.N` release links the source commit and merged PR, authored release notes, both Harbor artifacts, versioned pull references, and immutable digest references. `RELEASE` owns registry/project/platform metadata and the public Gitea `WEB_URL` for source/PR links; the built-in Gitea API URL may be internal. Update `HARBOR_PROJECT_ID` if moving projects. No personal token is required: only the release job requests repository-scoped write permission using the built-in Gitea job token. Harbor uses the existing repository Actions secrets.

A build failure leaves `latest` unchanged. Harbor cannot atomically promote two tags: a promotion failure can leave a mixed pair, with the release draft as the recovery marker. Fix the cause and rerun the same release workflow; retries reuse only tags belonging to the same source commit. A retry of an already published release is a no-op and does not rebuild or move version tags. Never overwrite a released version with different code or delete release history. To change code, merge a new PR/version. If a newer main exists, an older retry can publish its versioned release but skips latest promotion.

For rollback, use the prior release's manager and agent digest references through the normal deployment configuration/GitOps path. Review database/config compatibility first. Do not retag old images as a new release or claim a build proves a successful rollout.

## QA triage and completion

Use `qa`, `bug`, and `needs-triage` for incoming QA reports. Add one priority label (`priority/critical`, `priority/high`, `priority/normal`, `priority/low`) after assessing impact. Use `needs-reproduction` when details are missing and `ready-for-qa` when a published fix needs runtime confirmation. The issue records the failing release and the fixing PR/release. If `Fixes #N` closes it on merge, reopen it if QA still fails; use `Refs #N` when the issue should remain open through acceptance. Update Docmost with confirmed results and limitations, never secret values.

## Repository settings

Maintain squash-only merging, automatic branch deletion, issues/releases enabled, and a protected `main` with direct/force pushes disabled, stale branches blocked, rejected reviews blocking merge, no admin bypass, and required checks matching the actual Gitea job contexts. Branch protection is server-side and must be audited after repository migrations; the Markdown instructions alone do not enforce it. Publishing automation creates tags/releases, never bot commits to main.

## Build concurrency

The release Dockerfiles use `GOMAXPROCS=4` and `go build -p 2`, matching the Go validation job. These settings bound compilation within each manager/agent build stage; they do not set runtime container limits. The swarm05 resource-pressure incident remains tracked separately in #42 until its initiating cause is established.

## Deployment-owned updates

Control planes and detected Swarm tasks report `deploymentManaged: true`, link their running Gitea release, and leave image refresh to the service/stack deployment owner. The in-app container-recreation updater refuses these targets before pulling images or changing deployments. In this homelab, SwarmCD owns the stack and mutable image refresh.

Standalone release checks read the fork's published Gitea releases, including prereleases; draft and invalid tags are excluded. The repository currently requires authentication, so anonymous standalone polling is unavailable. No personal Git credentials are copied into Arcane. Dedicated private-feed authentication remains a follow-up; Gitea remains authoritative for release notes and artifacts.

## Volume helper companion

The release Dockerfiles also build ACFS from the pinned Constructorfleet Kit
module and package it beside the manager/agent binary. Keep its version and the
module replacement aligned. Volume operations install this immutable companion
into the helper's private `/tmp` filesystem, never the mounted volume. The tools
image continues to supply shell, archive and scanning utilities. No extra helper
privileges or execution-user changes are required.

The release Dockerfiles build and package `volume-stat` beside ACFS. It reads
filesystem capacity inside an existing helper's mounted volume. The usage probe
does not extend the helper's idle deadline.

Custom binary/static-image distributions should ship both Linux companions
beside the executable. Build `volume-stat` from `backend/cmd/volumestat` for the
Docker engine's architecture. Filesystem usage displays as unavailable if the
probe is missing. Without ACFS, legacy helper reads can work, but
existing-file writes require the ownership-preservation capability and fail
closed when it is absent. Run the volume tests and validate an owned local-file
edit's content, UID/GID and mode before marking workspace write QA accepted.
