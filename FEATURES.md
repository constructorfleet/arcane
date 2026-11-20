# Arcane feature checklist

Last reviewed: 2026-11-18.

This is the working scope for the Constructorfleet Arcane fork. Add, remove,
reword, or reprioritize entries here. IDs stay stable so changes can be requested
by ID. The detailed architecture is in [SWARM-FORK.md](SWARM-FORK.md).

A checked box means that specific statement has evidence. An upstream feature
is not considered complete until it works through our Swarm/global-agent setup.
“Configured” does not mean deployed. “Proposed” means an implementation approach
has been suggested but is still open for discussion.

## Working order

1. Control-plane separation and global agents.
2. Correct node routing, consoles, and plugin-volume file access.
3. Native resource onboarding and management.
4. Compose, shared inputs, registries, and identity verification.
5. Direct/Git deployment workflows, templates, observability, and release checks.

Change this ordering whenever priorities change.

## Confirmed groundwork

- [x] BASE-01 — Fork Arcane under Constructorfleet on GitHub and Gitea.
- [x] BASE-02 — Publish separate manager and agent images to Harbor.
- [x] BASE-03 — Remove implicit “local agent” coverage from the manager's Swarm node; service tests cover manager-node enrollment and duplicate protection.
- [x] BASE-04 — Provide a global-agent stack manifest accepted by Docker's stack parser.
- [x] BASE-05 — Configure the homelab Arcane role and remove Komodo's role and active inventory/integration references.
- [x] BASE-06 — Configure Harbor `latest` images and Arcane's SwarmCD per-stack image refresh.
- [x] BASE-07 — Validate manager-only bootstrap and agent-enabled role rendering with Ansible and Docker stack tooling.
- [x] BASE-08 — Create and verify an Arcane Docmost page describing actual status and limitations.
- [x] BASE-09 — Schedule separate hourly deployment-maintenance and documentation-maintenance follow-ups.
- [x] BASE-10 — Perform and verify the initial live Arcane rollout from homelab-ansible.
- [x] BASE-11 — Verify the live Authentik callback and administrator access.
- [x] BASE-12 — Enroll and verify one agent per Linux node, including the manager host.

## Control plane and global agents

- [x] CP-01 — Run the manager as a control plane with no Docker socket or local Docker daemon dependency. Implemented behind `ARCANE_CONTROL_PLANE=true`; socket-free HTTP smoke test passed.
- [ ] CP-02 — Remove the manager's implicit local-agent selection from startup, readiness, background jobs, and the UI. cf.42 targets remaining node-local automation registration and prevents control-plane Docker client acquisition; socket-free system backup execution remains unsupported and broader acceptance stays open (#102).
- [x] CP-03 — Deploy agents as one ordinary global Swarm service with no exclusion for the manager's host. Live deployment verified with five global agents, including the control-plane host.
- [ ] CP-04 — Keep each agent's connection, credentials, and node identity separate; do not route node operations through a round-robin `tasks.agent` address.
- [ ] CP-05 — Preserve agent identity across restarts, task replacement, and enrollment-token rotation without duplicate registrations.
- [ ] CP-06 — Select a verified, reachable manager agent for cluster operations and handle manager failover explicitly.
- [ ] CP-07 — Show offline, stale, mismatched, and missing agents accurately; never report them as healthy because the control plane is healthy.

Status: BASE-03 is service-tested. Socket-free deployment is verified with one
manager and five global agents on cf.6, including the manager's host. Directory
retirement and independent enrollment are verified; deletion of five agents was
verified in isolation. Node-agent association UI was verified in cf.8 (#5 closed);
credential rotation, failover, and complete node-local operation acceptance remain open.

## Node routing and interactive operations

- [x] NODE-01 — Resolve a service task to its node and container, then proxy requests to that node's agent.
- [x] NODE-02 — Open container consoles and exec sessions on the selected node and replica.
- [ ] NODE-03 — Stream container/task logs through the correct agent, with working disconnect and cancellation behavior.
- [ ] NODE-04 — Handle task movement and agent disconnection without silently switching an existing console to another replica.
- [ ] NODE-05 — Inspect and operate on node-local containers, images, volumes, and networks through the correct agent.

Status: cf.9 global and service task links resolved exact containers on swarm02,
swarm03, and swarm05 through their own agent UUIDs. Removed/unassigned tasks show
clear unavailable states (#8). A direct cf.15 shell session on the owned standalone
fixture on swarm01 returned its marker and exited cleanly (agent status 1000).
cf.22 service-task navigation and Shell markers passed on owned tasks on swarm02
and swarm03, with matching native container IDs and clean shell exits. A synthetic
stdout marker streamed through swarm03 Logs and Stop ended the stream. Both
nodes have manager role; worker-role, migration and failure/disconnect cases remain open.

## Volumes and files

cf.65 implements one logical row for same-name/driver Docker-global volumes in a known observed Swarm, with per-node records retained. It groups before filtering/pagination/counts and resolves the active application attachment before Workspace. Inactive or ambiguous groups do not initiate workspace access; unsaved drafts remain on their original agent. Coverage warnings and explicit physical-node links remain visible. cf.65 live QA verified both reported RBD volumes as one entry with five node observations, automatically routed to active swarm04 through manager01. Both root listings and15 nested migration files loaded; same-named local volumes stayed distinct, three pages/both driver sorts/cold reload passed, and native mappings/mountpoints stayed unchanged on all five nodes. Only two normal label-verified AutoRemove helpers appeared on active04. #13 is closed for this bounded logical grouping/routing; broader mutation, permissions and failover remain #33/#63.

cf.45 targets backend reference guards for global-volume delete/rename (#109), including declared zero-replica service mounts. Read-only production acceptance and broader storage identity/routing remain separate; no production volume mutation is claimed.

- [ ] VOL-01 — Browse plugin-backed volumes, including the homelab RBD plugin, through remote agents as well as the manager's node.
- [ ] VOL-02 — Mount volumes through Docker by name rather than assuming a host path for plugin mountpoints.
- [x] VOL-03 — Select the existing consumer's node for an attached RBD volume; verify attachment semantics before using another node. cf.65 verified unique active04 routing for both reported volumes; live migration/failover remains open.
- [ ] VOL-04 — Browse directories and download files with permissions and errors shown accurately.
- [ ] VOL-05 — Upload files, create directories, and edit/save files through the UI.
- [ ] VOL-06 — Support authorized rename and delete actions with clear target paths.
- [ ] VOL-07 — Protect file operations against path traversal, escaping symlinks, stale edits, and abandoned helper containers.
- [x] VOL-08 — Distinguish same-named node-local volumes and different Swarms/drivers. Group Docker-global volume observations within a known Swarm without claiming physical storage identity across clusters.
- [x] VOL-09 — Keep volume pagination and sorting usable when plugins repeat records; count each volume once per node and recover without clearing saved preferences.
- [x] VOL-10 — Preview supported raster images read-only, with explicit type/size limits and download states for unsupported files.
- [x] VOL-11 — Present one logical Docker-global volume per known Swarm/name/driver, retain physical-node observations and route its Workspace to a fresh unique active attachment. cf.65 #13 verified; physical identity across clusters is not inferred.

Status: the user confirmed RBD volume browsing on 2026-11-18. Full VOL-01
acceptance across remote and manager-host agents remains open. A cf.31 owned
local-file edit saved and reloaded its bytes but changed UID/GID from 501:10 to
0:0 (#86); the original bytes, owner and mode were restored. cf.33 packages an
ownership-preserving ACFS companion and rejects unsafe legacy updates. Live
cf.33 Save retained 501:10/0644 and the exact persisted 102 bytes on the owned
local fixture; native guards restored its original 60 bytes after browser
interruption. Cold UI reload of that cf.33 edited file was not completed.
On cf.34, a new disposable local file passed create/save, stale second-tab
rejection, cold reload and exact 64-byte download verification. A separate
53-byte upload retained its exact bytes through rename/move and reopen. Native
checks confirmed the original readme's bytes, 501:10/0644, inode and timestamps
were unchanged; guarded cleanup removed only the new files and their empty
folder. RBD/plugin writes, permission-denial and UI-delete acceptance remain open.
The pagination/sorting crash reported on 2026-11-18 was caused by duplicate
same-node volume names (`each_key_duplicate`). Release `3.0.1-cf.3` deduplicates
per daemon before pagination/counting. Its SwarmCD rollout and five-agent/manager
health were verified. Browser verification passed saved-preference
recovery, three pages, driver sorting in both directions, and navigation away/back.
Volume service and pagination tests and Go lint passed. Local format/lint was
blocked by missing `vp` at cf.3; later feature branches passed full local checks.

cf.17 local-fixture acceptance identified the exact agent-scoped Arcane volume
helper, retained In Use, and explained its mount ownership. Shared RBD identity,
grouping and safe attachment-aware navigation remain open in
[volume issue #13](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/13).

cf.36 checks driver-reported attachment metadata before workspace helper access.
Global plugin volumes reporting no local mountpoint fail promptly, including
cached-helper and fresh-create paths; unused local volumes remain accessible.
The user's inactive normalizer-state entry on swarm03 was read-only verified
as attached only on swarm04. No inactive RBD mount was attempted. The guard
does not discover backend identity or automatically route to another agent;
shared grouping/navigation and broader RBD acceptance remain #13/#14/#33.

Live cf.36 reached a prompt unavailable state on the supplied inactive swarm03
URL, but its generic conflict category was hidden as an internal error by the
workspace-specific HTTP boundary. cf.37 uses the workspace conflict category;
the existing inactive cached/fresh helper cases now assert that exact contract.
Corrected live UI and no-helper acceptance remain tracked in #14.

## Native Docker resources and onboarding

- [ ] RES-01 — Discover resources on an existing Swarm regardless of whether Ansible, SwarmCD, Docker CLI, Arcane, or another UI deployed them.
- [ ] RES-02 — Manage existing stacks/services without recreating or relabeling them merely to onboard them.
- [ ] RES-03 — Inspect services/tasks and edit native service settings, replicas, constraints, labels, mounts, ports, and update policies.
- [ ] RES-04 — Launch standalone containers from the UI on an explicitly selected node.
- [ ] RES-05 — Create, inspect, and configure Docker networks, including relevant overlay/attachable/network settings.
- [ ] RES-06 — Inspect and manage volumes, including their drivers and driver options.
- [ ] RES-07 — Inspect nodes, manage labels, promote/demote roles, and change active/pause/drain availability.
- [ ] RES-08 — Inspect effective container/service environment variables with appropriate access controls.
- [ ] RES-09 — Distinguish discovered runtime state, reconstructed Compose, and original deployment source.

Status: several operations exist upstream. Each needs verification in our
control-plane/global-agent architecture before being marked complete.

## Compose and stack authoring

- [x] CMP-01 — Create and edit stacks directly in the UI using standard Compose YAML.
- [ ] CMP-02 — Parse and validate Compose, then validate the supported feature set for the selected target: Swarm stack or local Compose.
- [ ] CMP-03 — Preserve useful YAML anchors, merges, extension fields, and Arcane's existing authoring conveniences.
- [ ] CMP-04 — Audit and retain useful Arcane custom tags/features with documented expansion to deployable Compose. **Candidate features; exact set undecided.**
- [ ] CMP-05 — Show resolved nonsecret variables, effective images, validation errors, and the deployment diff before applying changes.
- [ ] CMP-06 — Keep GUI editing and YAML editing consistent, and warn when an edit cannot preserve a document feature.
- [ ] CMP-07 — Preserve source Compose, associated files, deployment history, and enough provenance to explain live drift.
- [x] CMP-08 — Launch standalone containers from the UI through an explicitly selected node agent. cf.23 owned swarm03 launch and native command/resource/security settings matched; cf.26 also accepted 0.1 CPU creation and 0.25 CPU editing with exact native NanoCPUs (#69 closed).
- [x] CMP-09 — Preserve full current native service Spec, resource IDs, network and source bytes/permissions on an unchanged-source UI Save. cf.68 owned zero-task acceptance passed (#144); general Compose and running-consumer round trips remain open.
- [ ] CMP-10 — Navigate from a project's files/services to its container without a route-data error. cf.69 reproduced undefined.data during the client transition; cold reload succeeded (#150). Captured mounted-page cache keys pass the isolated transition; live acceptance remains pending.

cf.69 project lifecycle acceptance: ordinary UI Create→Up→Check→Destroy produced exact owned image-check counts 0→1→0 and removed the project, source, container and private bridge (#167 closed/verified). Read-only exact-project SQLite counts required no manual cleanup. All19 core Specs and40 task IDs matched the accepted69 baseline after cleanup; no fixture remains. This does not sweep existing orphan records or establish concurrent-check cleanup.

Status: cf.23 owned Swarm stack create/Edit/Save passed through explicitly selected
swarm03. Original anchor/merge and synthetic `.env` input survived Source refresh;
native runtime environment expansion, metadata-label update, 0.1 CPU, 32 MiB and
read-only/capability settings matched. The owned service/network and source files
were cleaned up. Advanced/unsupported-feature validation, deployment previews,
history and broader GUI/source round trips remain open.

Boundary: no required TOML or replacement service/stack definition language.
Swarm does not support every modern local-Compose feature; unsupported behavior
must be surfaced rather than silently ignored.

## Configs, secrets, and reusable inputs

cf.67 retains desired config/secret revisions and all current, rollback and retained-task dependencies during automatic cleanup; failed reference inventory defers removal. Owning SDK coverage and live zero-task A→B PreviousSpec retention passed (#147). On cf.68, an actual unchanged-source Edit→Save preserved the complete service Spec, resource IDs, raw network and source bytes/permissions (#144). Both owned fixtures were cleaned up normally. The first fixture's external agent reference/task churn remains #166; all19 core Specs/task IDs stayed unchanged during the separate cf.68 fixture. No live rollback, running-consumer rotation or secret plaintext readback is claimed.

- [ ] SEC-01 — Create Docker configs in the UI and inspect their content and usage.
- [ ] SEC-02 — Create Docker secrets in the UI and inspect metadata and consumers.
- [ ] SEC-03 — Edit/replace configs through immutable revisions and update selected consumers.
- [ ] SEC-04 — Replace/rotate secrets through immutable revisions and update selected consumers.
- [ ] SEC-05 — Define reusable global variables and secrets, with clear scope and stack overrides.
- [ ] SEC-06 — Reference shared values from multiple stacks without copying plaintext into each Compose document.
- [ ] SEC-07 — Read and edit authorized variable values while keeping secret values out of ordinary logs, metrics, diffs, and exports.
- [ ] SEC-08 — Preview which stacks a shared-value change affects and select which deployments to update.
- [x] SEC-09 — Retain referenced config/secret revisions and defer automatic removal on incomplete reference inventory. cf.67 owning checks and live rollback-reference retention passed; broader rotation and reusable-input scope remain open.

Status: native config/secret counts, two next-page transitions and config name
sorting passed a read-only cf.19/cf.20 inventory check. UI creation through swarm03
produced an owned native config rediscovered through swarm01. cf.21 list and
aggregate responses omitted its content, while authorized inspection/UI display
and a label update retained the original marker bytes. cf.22 synthetic secret
creation through explicitly selected swarm03, swarm01 discovery, immutable-value
guidance and persisted label editing also passed; native metadata matched. Both
unused fixtures were removed. Negative scoped-role cases remain #63; immutable
revisions, consumer management and reusable-global delivery are still open.

Reusable-input source audit at cf.28: existing global-variable service encrypts
secret-marked storage and redacts mutation/list DTOs; omitted values preserve
ciphertext, and secret-to-readable changes require a replacement value.
Agent-scoped values override all-agent values; interpolation-only inputs need
not be injected into containers. Control-plane synchronization skips local0.
These effective values materialize as agent-side `.env.global` inputs, distinct
from native Swarm secret delivery. Existing variable/project business coverage
passed PR #75 Go CI; live synthetic reuse and scoped-denial acceptance remain
#29 / #63. No production values were read or changed for this audit.

cf.39 connects Swarm render/deploy/redeploy to the selected agent's synchronized
shared inputs (#92). Explicit stack values, including blank values and nested
expansion, override defaults; interpolation-only inputs remain available to
explicit references. Secret provenance survives synchronization; legacy inputs
require a successful resync before Swarm reuse. Imports from agents lacking
provenance default to secret storage; existing manager definitions are preserved.
Protected previews withhold
resolved YAML and resource names, and protected errors/path warnings omit
values. Saved source retains original Compose and explicit .env. Scope-aware
completion offers secret keys without values. Isolated fake-Docker business
coverage and frontend checks are distinguished from post-release acceptance in
#92; SEC-05–SEC-08 and scoped-denial acceptance remain open. Inputs are the last
successfully synchronized agent snapshot; explicit Docker-visible substitutions
remain subject to Docker permissions. Native secrets are a separate mechanism.

Boundary: Docker cannot return the plaintext of an existing Swarm secret.
Imported secrets must be shown as external/write-only; replacement is supported
without pretending their old values can be recovered. Standalone containers
need a distinct secret-delivery mechanism because Swarm secrets are service-only.

## Identity and registries

- [x] AUTH-01 — Verify OIDC sign-in through Authentik, including the deployed callback and group/role mapping.
- [ ] AUTH-02 — Enforce permissions on backend requests and interactive streams, including exec, file writes, deployment, and secret operations.
- [ ] REG-01 — Pull private images with configured third-party registry credentials (Harbor in this homelab) during service creation, updates, standalone launches, and helper-container creation.
- [ ] REG-02 — Ensure newly scheduled tasks on uncached nodes can pull authenticated images.
- [ ] REG-03 — Support explicitly configured image-rewrite policies for pull-through caches, following the homelab-ansible pattern.
- [ ] REG-04 — Preserve image paths, tags, and digests; normalize Docker Hub official images correctly; avoid rewriting configured cache references recursively, including Harbor.
- [ ] REG-05 — Show original and effective image references and allow policy opt-outs where needed.
- [x] REG-06 — Push an existing node image tag through its explicitly selected owning agent, with registry credentials, progress and actual errors.
- [x] REG-07 — Build a saved local workspace through an explicitly selected agent and load its resulting image on that node.

Status: fresh private-window OIDC sign-in/logout/callback and the expected SSO
Administrator role passed on cf.20; the original browser session remained active.
The callback was already live and needed no Terraform apply. Alternate groups,
role removal and scoped resource/stream/file allow/deny acceptance remain in
[AUTH-02 issue #63](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/63).

cf.18 verified registry connection, accurate catalog-denial wording, known-repository
browsing and an explicitly selected swarm03 tag pull. Before/after inventories
proved the destination; image layers were already cached. Private uncached pulls,
non-Harbor runtime and broader build/push acceptance remain
[registry issue #40](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/40).
cf.22 standalone Push passed for a unique owned QA tag from explicitly selected
swarm03. Activity output, the native agent request and Harbor artifact metadata
agree on the pushed digest. The new repository/tag browses in Arcane and a
subsequent explicit swarm02 Pull passed. Layers were cross-mounted or cached;
that cf.22 check did not establish uncached transfer. Negative and
cancellation behavior has business-test coverage. No released tag was overwritten.

cf.24 UI-created synthetic Dockerfile/text built through selected swarm03, then a
separate Push transferred a new layer to Harbor. Selected swarm02 Pull acquired
that previously absent layer and matched its native digest/label. Both context
files survived cf.25 agent replacement with exact hashes and reopened in the UI;
the exact owned files were then cleaned up. The dedicated node-local `/builds`
mount is separate from identity state. Repository deployment-example prevention shipped in cf.27 / PR #74,
closing #71. Private authentication, non-Harbor providers, integrated Build+Push,
and broader cache/provider behavior remain open.

## Direct deployment and GitOps

- [ ] GIT-01 — Support directly deployed stacks with versioned Compose/config/secret input bundles.
- [ ] GIT-02 — Support repository-managed stacks whose Compose and associated inputs can be edited in the UI and published to Git.
- [ ] GIT-03 — Preserve compatibility with homelab base stacks and GitOps stacks rather than adding a competing resource model.
- [ ] GIT-04 — Establish one reconciliation owner per stack, including external Ansible/SwarmCD ownership.
- [ ] GIT-05 — Display desired Git revision, last applied revision, reconciliation errors, and live drift.
- [ ] GIT-06 — Make emergency live edits explicit and provide a way to carry the change back to deployment source.
- [ ] GIT-07 — Export or reference secrets safely; keep SOPS-compatible encrypted files for the existing homelab GitOps path.

Proposed approach: direct and repository-managed stacks share a deployment
pipeline and bundle format. Whether UI Git changes push directly, open PRs,
or support both remains an explicit design decision.

## Templates, metrics, and operations

- [ ] TPL-01 — Support reusable templates with declared variables/secret inputs and a preview before deployment.
- [ ] TPL-02 — Audit and reuse Arcane's existing template/catalog behavior where it fits standard Compose.
- [ ] OBS-01 — Expose Prometheus metrics for manager health, agent connectivity, operations, errors, deployment results, and reconciliation state.
- [ ] OBS-02 — Provide useful UI error/status information for partial deployment failures and unavailable nodes.
- [x] OBS-03 — Collect existing manager and node-agent Prometheus series through six unique task targets on the monitoring overlay, with remote-query and native container-ID coverage proof.
- [ ] OPS-01 — Record an audit trail for mutations and sensitive operations without recording secret payloads.
- [ ] OPS-02 — Verify rollback, backup/restore, encryption-key preservation, and upgrades for manager state.
- [ ] OPS-03 — Keep Arcane Docmost feature, customization, deployment, and troubleshooting pages synchronized with verified changes.

Template evidence: a cf.15 unsaved Compose anchor/merge draft and matching
synthetic `.env` input passed editor diagnostics; invalid YAML produced an error.
Existing metadata business tests also pass for anchors, interpolation, includes,
name/color tags and global environment loading. Existing template-service tests
pass for downloaded/local icon preservation and refusing unowned overwrites.
cf.22 saved/reopened an owned template after refresh with literal anchors/merge,
extension metadata and synthetic `.env` retained. An explicit interpolated icon
URL rendered in the gallery. cf.25 resolved the retained `nginx` catalog slug
without editing or resaving, closing #67.
cf.28 derived both variant URLs from the retained variant-only template without
editing/resaving; the gallery and picker rendered nginx artwork in the existing
dark theme, closing #73. Its unsaved project draft retained anchors/merge and both
source variants, then was closed without creating a project. No template stack
was deployed; deployment interpolation, extension application, separate theme-switch
visual acceptance and final branding remain open in
[template issue #17](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/17).

Administration review remains #16. The parent Connections stream passed bounded
Pause/Resume/Refresh and owned-page teardown (3 → 6 → 3 connections). On cf.34,
Actors continued updating while the parent header was paused (#93). cf.38 passes
page pause state to the independent Actors/log views, retains the latest actor
snapshot and a bounded incoming log buffer, and preserves explicit Clear and
reconnect behavior. Six source-callback/effect cases pass with framework
scheduling mocked; the existing log browser case was extended but development
E2E was not run. Post-release live acceptance is tracked in #93; broader
administration and scoped-negative cases remain open.

Metrics: the existing OTel path supports a separate Prometheus exporter and
instruments agent/tunnel, container, Swarm, HTTP, Git sync, backup and
runtime/database activity. Unset exporters default off. Homelab PR #12 enabled
private collection for one manager and five agents; repeated target and
VictoriaMetrics queries passed, and sampled container IDs matched native Docker
on all five nodes. Exporters listen on attached overlays with no new host port
or public router. Immediate warm and successful replacement scrapes passed;
fully isolated cold-cache completeness remains unproven. Broader operation/
reconciliation coverage and dashboards/alerts remain
[metrics issue #32](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/32).

During that configuration rollout, node05 recovered naturally from an OOM and
pressure incident; manager relocation preserved all five agent UUIDs and data
fencing. RBD handoff evidence remains #42. A source/heap audit identified 8 GiB
of fixed collector cardinality-tracking allocations; the tracking-policy choice
and current mitigation status are recorded in
[infrastructure issue #76](https://gitea.home.prettybaked.com/constructorfleet/arcane/issues/76).
These observations do not establish the initiating allocation of the global OOM.

## Decisions still open

- [ ] DEC-01 — Confirm the exact custom tags/authoring conveniences worth keeping.
- [ ] DEC-02 — Choose Git publication modes: direct commits, PRs, or both, and their rollout gates.
- [ ] DEC-03 — Define the global variable/secret precedence rules and secret source-of-truth options.
- [ ] DEC-04 — Confirm the RBD plugin's attachment policy and behavior for unused volumes.
- [ ] DEC-05 — Decide whether multi-replica control-plane HA belongs in the initial release or a later milestone. **Proposed follow-up, not a committed initial requirement.**

## First useful release acceptance

- [x] ACCEPT-01 — Manager runs without a Docker socket; healthy readiness does not require a local Docker daemon.
- [x] ACCEPT-02 — A global agent runs once on every eligible node, including the manager's host.
- [ ] ACCEPT-03 — Console/exec reaches the selected task on either of two different workers.
- [ ] ACCEPT-04 — RBD volume browsing, upload, and editing work on the correct node without unsafe additional attachment.
- [ ] ACCEPT-05 — Existing externally deployed resources appear and can be operated on without redeployment during discovery.
- [ ] ACCEPT-06 — OIDC, private-image pulls, Prometheus output, and GitOps update ownership are demonstrated in the target deployment.

## Where to edit

This repository checklist is the source for scope and progress. Docmost contains
operator-facing documentation, including
[the Arcane page](https://docmost.home.prettybaked.com/s/general/p/arcane-hkk6Tx4CHx).
Changes to scope should be reflected here before the documentation agent presents
them as committed behavior. Checking off a feature should cite its implementation
and relevant verification in the change that completes it.

Stack source-persistence follow-up (#88 / CMP-07): an isolated storage failure
previously allowed fake Docker network/service creation and a success response
without saved source. cf.34 returns the persistence error before deployment.
The negative case and related stack-source/Swarm business coverage pass; normal
release rollout acceptance and broader source/GitOps behavior remain separate.
