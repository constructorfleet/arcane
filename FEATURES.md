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
- [ ] BASE-10 — Perform and verify the initial live Arcane rollout from homelab-ansible.
- [ ] BASE-11 — Verify the live Authentik callback and administrator access.
- [ ] BASE-12 — Enroll and verify one agent per Linux node, including the manager host.

## Control plane and global agents

- [x] CP-01 — Run the manager as a control plane with no Docker socket or local Docker daemon dependency. Implemented behind `ARCANE_CONTROL_PLANE=true`; socket-free HTTP smoke test passed.
- [ ] CP-02 — Remove the manager's implicit local-agent selection from startup, readiness, background jobs, and the UI.
- [x] CP-03 — Deploy agents as one ordinary global Swarm service with no exclusion for the manager's host. Live deployment verified with five global agents, including the control-plane host.
- [ ] CP-04 — Keep each agent's connection, credentials, and node identity separate; do not route node operations through a round-robin `tasks.agent` address.
- [ ] CP-05 — Preserve agent identity across restarts, task replacement, and enrollment-token rotation without duplicate registrations.
- [ ] CP-06 — Select a verified, reachable manager agent for cluster operations and handle manager failover explicitly.
- [ ] CP-07 — Show offline, stale, mismatched, and missing agents accurately; never report them as healthy because the control plane is healthy.

Status: BASE-03 is service-tested. Socket-free deployment is verified with one
manager and five global agents on cf.6, including the manager's host. Directory
retirement and independent enrollment are verified; deletion of five agents was
verified in isolation. Node-agent association UI is tracked separately in #5;
credential rotation, failover, and complete node-local operation acceptance remain open.

## Node routing and interactive operations

- [x] NODE-01 — Resolve a service task to its node and container, then proxy requests to that node's agent.
- [ ] NODE-02 — Open container consoles and exec sessions on the selected node and replica.
- [ ] NODE-03 — Stream container/task logs through the correct agent, with working disconnect and cancellation behavior.
- [ ] NODE-04 — Handle task movement and agent disconnection without silently switching an existing console to another replica.
- [ ] NODE-05 — Inspect and operate on node-local containers, images, volumes, and networks through the correct agent.

Status: cf.9 global and service task links resolved exact containers on swarm02,
swarm03, and swarm05 through their own agent UUIDs. Removed/unassigned tasks show
clear unavailable states (#8). Console/exec, logs, migration, and disconnect
acceptance remain open.

## Volumes and files

- [ ] VOL-01 — Browse plugin-backed volumes, including the homelab RBD plugin, through remote agents as well as the manager's node.
- [ ] VOL-02 — Mount volumes through Docker by name rather than assuming a host path for plugin mountpoints.
- [ ] VOL-03 — Select the existing consumer's node for an attached RBD volume; verify attachment semantics before using another node.
- [ ] VOL-04 — Browse directories and download files with permissions and errors shown accurately.
- [ ] VOL-05 — Upload files, create directories, and edit/save files through the UI.
- [ ] VOL-06 — Support authorized rename and delete actions with clear target paths.
- [ ] VOL-07 — Protect file operations against path traversal, escaping symlinks, stale edits, and abandoned helper containers.
- [ ] VOL-08 — Distinguish same-named node-local volumes; do not assume a name or driver's “global” scope proves shared storage identity.
- [x] VOL-09 — Keep volume pagination and sorting usable when plugins repeat records; count each volume once per node and recover without clearing saved preferences.
- [x] VOL-10 — Preview supported raster images read-only, with explicit type/size limits and download states for unsupported files.

Status: the user confirmed RBD volume browsing on 2026-11-18. Full VOL-01
acceptance across remote and manager-host agents remains open; uploading and
editing remain unverified. The pagination/sorting crash reported on 2026-11-18 was caused by duplicate
same-node volume names (`each_key_duplicate`). Release `3.0.1-cf.3` deduplicates
per daemon before pagination/counting. SwarmCD rolled out both images; all five
agents and the manager are healthy. Browser verification passed saved-preference
recovery, three pages, driver sorting in both directions, and navigation away/back.
Volume service and pagination tests and Go lint passed. Full local format/lint
remains blocked by missing `vp`; the manager image frontend build passed.

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

- [ ] CMP-01 — Create and edit stacks directly in the UI using standard Compose YAML.
- [ ] CMP-02 — Parse and validate Compose, then validate the supported feature set for the selected target: Swarm stack or local Compose.
- [ ] CMP-03 — Preserve useful YAML anchors, merges, extension fields, and Arcane's existing authoring conveniences.
- [ ] CMP-04 — Audit and retain useful Arcane custom tags/features with documented expansion to deployable Compose. **Candidate features; exact set undecided.**
- [ ] CMP-05 — Show resolved nonsecret variables, effective images, validation errors, and the deployment diff before applying changes.
- [ ] CMP-06 — Keep GUI editing and YAML editing consistent, and warn when an edit cannot preserve a document feature.
- [ ] CMP-07 — Preserve source Compose, associated files, deployment history, and enough provenance to explain live drift.

Boundary: no required TOML or replacement service/stack definition language.
Swarm does not support every modern local-Compose feature; unsupported behavior
must be surfaced rather than silently ignored.

## Configs, secrets, and reusable inputs

- [ ] SEC-01 — Create Docker configs in the UI and inspect their content and usage.
- [ ] SEC-02 — Create Docker secrets in the UI and inspect metadata and consumers.
- [ ] SEC-03 — Edit/replace configs through immutable revisions and update selected consumers.
- [ ] SEC-04 — Replace/rotate secrets through immutable revisions and update selected consumers.
- [ ] SEC-05 — Define reusable global variables and secrets, with clear scope and stack overrides.
- [ ] SEC-06 — Reference shared values from multiple stacks without copying plaintext into each Compose document.
- [ ] SEC-07 — Read and edit authorized variable values while keeping secret values out of ordinary logs, metrics, diffs, and exports.
- [ ] SEC-08 — Preview which stacks a shared-value change affects and select which deployments to update.

Boundary: Docker cannot return the plaintext of an existing Swarm secret.
Imported secrets must be shown as external/write-only; replacement is supported
without pretending their old values can be recovered. Standalone containers
need a distinct secret-delivery mechanism because Swarm secrets are service-only.

## Identity and registries

- [ ] AUTH-01 — Verify OIDC sign-in through Authentik, including the deployed callback and group/role mapping.
- [ ] AUTH-02 — Enforce permissions on backend requests and interactive streams, including exec, file writes, deployment, and secret operations.
- [ ] REG-01 — Pull private images with Harbor credentials during service creation, updates, standalone launches, and helper-container creation.
- [ ] REG-02 — Ensure newly scheduled tasks on uncached nodes can pull authenticated images.
- [ ] REG-03 — Support explicitly configured image-rewrite policies for pull-through caches, following the homelab-ansible pattern.
- [ ] REG-04 — Preserve image paths, tags, and digests; normalize Docker Hub official images correctly; avoid rewriting Harbor references recursively.
- [ ] REG-05 — Show original and effective image references and allow policy opt-outs where needed.

Status: upstream OIDC/registry features and homelab callback configuration exist;
end-to-end behavior still needs verification.

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
- [ ] OPS-01 — Record an audit trail for mutations and sensitive operations without recording secret payloads.
- [ ] OPS-02 — Verify rollback, backup/restore, encryption-key preservation, and upgrades for manager state.
- [ ] OPS-03 — Keep Arcane Docmost feature, customization, deployment, and troubleshooting pages synchronized with verified changes.

## Decisions still open

- [ ] DEC-01 — Confirm the exact custom tags/authoring conveniences worth keeping.
- [ ] DEC-02 — Choose Git publication modes: direct commits, PRs, or both, and their rollout gates.
- [ ] DEC-03 — Define the global variable/secret precedence rules and secret source-of-truth options.
- [ ] DEC-04 — Confirm the RBD plugin's attachment policy and behavior for unused volumes.
- [ ] DEC-05 — Decide whether multi-replica control-plane HA belongs in the initial release or a later milestone. **Proposed follow-up, not a committed initial requirement.**

## First useful release acceptance

- [ ] ACCEPT-01 — Manager runs without a Docker socket; healthy readiness does not require a local Docker daemon.
- [ ] ACCEPT-02 — A global agent runs once on every eligible node, including the manager's host.
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
