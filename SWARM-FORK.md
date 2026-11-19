# Swarm management fork

## Status

Starting point: Arcane main `79c61a2b6`, reporting version 3.0.1.
Fork: https://github.com/constructorfleet/arcane.
Build repository: https://gitea.home.prettybaked.com/constructorfleet/arcane.

This is the implementation contract for the fork, not a claim that the target
system is complete. AI assistance: OpenAI Codex performed source inspection,
architecture drafting, and implementation. Production verification as of
2026-11-18: cf.3 runs a socket-free control plane and five independent global
agents; the user confirmed RBD browsing, and pagination/sort recovery was
verified after the duplicate-volume fix. File mutation and full all-node
acceptance remain open. See [FEATURES.md](FEATURES.md) for scope and
[RELEASING.md](RELEASING.md) for branches, squash PRs, releases, and QA intake.

The initial patch removes implicit local-agent coverage from Swarm nodes. A
manager's own node must enroll an agent through the same path as workers. Agent
coverage is recalculated by the control plane rather than trusting an upstream
node response's local-agent status. Existing duplicate and identity-mismatch
protections remain in effect. The adjacent service tests exercise enrollment
and coverage for manager and worker nodes.

The global agent manifest is an example. The homelab Ansible role owns the
actual GitOps deployment, including manager state and agent enrollment. Its
global deployment was verified at five of five nodes, including the manager
host; node-local identity storage survives task replacement.

Set `ARCANE_CONTROL_PLANE=true` on the manager to run without a Docker socket.
This mode retires the old local-agent directory entry without deleting its
stored data, rejects local-engine resource routes, skips local Docker event and
refresh workers, and checks database readiness independently of Docker. Agents
remain in ordinary `ARCANE_AGENT=true` mode.

Control-plane enrollment creates a visible, independent identity for every node
without first requiring another manager agent to be connected. This removes
cluster-parent discovery as a bootstrap dependency. Registered node agents,
including the node hosting the manager, can all be deleted through the ordinary
agent API. Deletion revokes their credentials; redeployment/re-enrollment is
needed to bring removed agents back.

Verified with service tests and an isolated running manager using a nonexistent
Docker socket: readiness reports database health, no local agent appears, five
nodes enroll with one unlimited token, and all five registrations can be deleted.
Real-node global deployment and initial RBD browsing are also confirmed;
console access, file mutations, and complete cross-node acceptance still need
separate verification. Legacy manager-local behavior remains available when this option
is not enabled, to permit a deliberate migration of existing installations.

## Initial verification

- `go test ./internal/swarm/... ./internal/enrollment/...`: all seven packages passed.
- `docker stack config --compose-file docker/examples/stack.agents.yaml`: passed
  with placeholder image and manager URL, without deploying anything.
- `gofmt` on both changed Go files and `git diff --check`: passed.
- Required `just format all` and `just lint all`: attempted, blocked by missing
  Vite Plus (`vp`) and `golangci-lint` tooling. Full checks have not passed.
- At this initial patch stage, browser/runtime/Swarm/RBD integration was unverified; current verification is recorded in Status above and FEATURES.md.

The editable feature scope and progress checklist is [FEATURES.md](FEATURES.md).

## Control-plane verification (2026-11-18)

- Domain tests passed for environments, enrollment, Swarm, health, middleware,
  variables, configuration/schema, GitOps, bootstrap, and dependency wiring.
- Go formatting and lint passed for backend, CLI, and shared types using the
  existing `exclude_frontend` build tag for backend checks.
- A running isolated control plane with a nonexistent Docker socket passed
  `/api/health/ready`, listed no local agent, enrolled five independent node
  identities with one unlimited token, and accepted deletion of all five.
- Full workspace formatting/lint still requires the missing Vite Plus (`vp`)
  tool. No frontend source was changed in this increment.

## Architecture decisions

### Control plane and agents

The manager runs the UI, authentication, authorization, agent directory, desired
state, deployment records, and audit log. It needs no Docker socket, host mounts,
or privileged mode. It may run on the managed Swarm or outside it. Initial
control-plane deployment can use one replica with persistent state; manager
replication and leader election are separate work, not an implied capability.

Deploy one agent on every supported node using a global Swarm service, including
all manager nodes. No exclusion of the node hosting the control plane. Agents
connect outbound using the existing authenticated tunnel transport. Bootstrap
uses a revocable, appropriately limited enrollment token; each enrolled agent
retains its own credentials in node-local persistent storage. Validate agent
identity against Docker node and cluster identities before activating a binding.
Handle simultaneous first enrollment, reconnects, token rotation, and task
replacement without creating duplicate environments.

Cluster identity and node identity must be separate from transport addresses.
Key node bindings by cluster ID and node ID. A hostname, IP address, container
ID, or DNS round-robin result is not a node's identity. Move cluster membership
away from ownership by one particular parent-agent environment; agents changing
role or losing connectivity must not orphan the cluster.

Cluster operations select a reachable, verified manager agent with appropriate
capabilities. Use Docker object versions for updates. On ambiguous timeout,
inspect the outcome before retrying a mutation. Reads may fail over to another
manager. Do not replay exec, uploads, or mutations against another node.

Node operations explicitly target one agent: container exec/attach/console,
container filesystem, local images, node networks, and volume files. Resolve a
Swarm task through its node ID and container ID, then pin the stream to that
agent. Task rescheduling ends the old console; offer a fresh session on the new
task. Never silently attach to a different replica. Missing agents produce a
clear unavailable state.

`tasks.agent` may assist discovery. It cannot be the routing decision for a
node-specific request. The browser talks to the control plane; the control plane
authorizes and proxies to the selected agent, including WebSocket/stream flow
control, cancellation, and session revocation.

### Plugin volumes and files

Reuse Arcane's helper-container mechanism. Ask the selected node's Docker Engine
to mount the named volume at a fixed path inside the helper. Do not open Docker's
reported mountpoint on the manager, assume `/var/lib/docker/volumes`, or bind a
plugin's private filesystem into the manager.

Volume names alone are not globally unique storage identities. Keep observations
per node and driver. A driver's advertised global scope is not proof of safe
concurrent mounts or filesystem identity across nodes.

For an in-use RBD volume, browse on the node already consuming it. Do not attach
it elsewhere as a side effect of opening the browser. Read-only filesystem access
can still cause a block-device attachment, so read-only alone is insufficient.
For unused volumes, require an explicit node selection and verified attachment
policy. Show unavailable, conflicting, or unknown ownership rather than guessing.
Inspect the actual RBD plugin before defining its attachment policy.

Default to read-only sessions; authorize writes separately. Use temporary files
and atomic replacement where supported, preserve permissions, check the version
or content hash on save, and reject stale edits. Enforce path containment for
symlinks and archive uploads inside the mounted volume. Bound upload sizes and
clean up helpers on disconnect, timeout, and agent restart. Test that removing a
helper never deletes a user's volume. Existing helper code needs review against
these requirements before claiming RBD support.

### Docker remains the resource model

Discover every Docker resource visible through the agents, regardless of creator:
services, tasks, containers, stack labels, networks, volumes, configs, secrets,
node roles, availability, labels, and registry/plugin capabilities. Deduplicate
cluster-wide resources while retaining node-local observations.

Discovery never deploys, recreates, relabels, or takes ownership of a resource.
Unmanaged resources remain operable through native Docker operations. Existing
stacks can be adopted deliberately. Docker service state cannot reconstruct the
original Compose, anchors, comments, environment inputs, source files, or secret
payloads. Generated Compose must be marked as reconstructed and reviewed before
becoming deployment source.

Store original Compose and deployment provenance for UI-managed stacks. Offer a
GUI and YAML editor over the same document, preserving unknown supported fields
and warning before an editor cannot round-trip a feature. Do not invent a second
service/stack definition language or require TOML.

### Compose and deployment

Use a maintained Compose parser and schema for parsing, anchors, interpolation,
and standard extension fields. Then perform target-specific validation: Docker
Swarm stack deploy supports a different feature set from local Docker Compose.
Do not silently ignore unsupported deployment semantics. Standalone containers
must select a node, since Swarm does not schedule standalone containers.

Retain useful Arcane variable/template features after auditing their semantics.
Prefer `x-` extension fields for optional application features. Any custom YAML
tag must have a documented expansion and export to plain deployable Compose.
Never evaluate arbitrary code during template expansion. Record source revision,
resolved nonsecret inputs, rewritten images, and resulting Docker object IDs.

Show the deployment diff before applying it and expose partial failures and
rollback state. Serialize conflicting operations on the same stack. Rollback
must preserve required old config/secret revisions; garbage collection follows
reference checks rather than deleting everything with an old name.

### Configs, secrets, and shared inputs

Support config and secret creation, references, labels, usage inspection, and
rotation in the UI. Their payloads are immutable in Swarm: editing creates a new
revision and updates selected consumers. Existing secrets are write-only because
Docker inspection does not return their values. Never imply that importing a
cluster recovers secret contents.

Provide a reusable variable and secret catalog with explicit global, cluster,
and stack scope and deterministic precedence. Preview resolved nonsensitive
values and secret reference names. Keep deployment-time interpolation distinct
from runtime mounted configs/secrets. Shared-secret rotation previews affected
stacks and deploys only chosen consumers.

If the application retains source secrets for reuse, encrypt them separately
from Docker, with a managed encryption key and recovery/rotation procedure.
Use references for external secrets when source values need not be retained.
Protect values from logs, metrics, diffs, frontend caches, and Git. Standalone
containers do not natively support Swarm secrets; offer a separately documented
secret delivery mechanism rather than pretending they do.

### Direct and repository-managed stacks

One deployment pipeline consumes an immutable bundle of Compose, config files,
secret references/encrypted files, and resolved image references. Stack source is
explicitly direct or repository-managed, with exactly one reconciliation owner.

Direct mode saves a versioned bundle in the application and deploys it. Git mode
edits the bundle in a branch and creates a commit or PR. Deploy only the accepted
revision using the designated reconciler. Support existing SwarmCD ownership;
do not also run an independent Arcane reconciliation loop for that stack.

Model source mode separately from origin: a stack deployed by Ansible can be
observed and operated without transferring ownership from Ansible. UI edits to
externally managed resources show drift and identify the source to update.
Emergency live changes in Git mode are explicit, audited, and can be exported
back to Git. Display desired revision, last applied revision, live state, and
reconciliation errors.

The homelab bundle layout is a useful default:

- `<stack>/compose/compose.yml`
- `<stack>/files/<config-name>`
- `<stack>/secrets/<secret-name>` (SOPS encrypted)

Retain compatibility with base stacks referencing precreated external objects
and GitOps stacks using file-backed configs/secrets. Config/secret versioning
must follow the selected deployment owner's policy. Git export of imported
secrets requires replacement values or external references, never placeholders
silently deployed as real secrets.

### Registries, identity, templates, and telemetry

Reuse existing OIDC, registry credentials, templates, authorization, and telemetry
where they fit. Audit their behavior through the agent path before treating an
upstream checkbox as a completed fork feature.

Registry rewriting is an explicit deployment policy independent of registry
runtime configuration. Follow the existing homelab mapping: Docker Hub official
images gain `library/`; explicit upstream hosts map to Harbor proxy projects;
tags and digests survive unchanged; Harbor's own references are not recursively
rewritten. Show original and effective references and support opt-outs. Resolve
credentials for the effective registry and supply registry auth on Swarm service
creation and update, so workers and rescheduled tasks can pull images. Helper
images and standalone pulls need the same policy and credentials.

OIDC authenticates users; backend permissions authorize every operation and
stream. Separate resource viewing, exec, filesystem read/write, deployment,
secret replacement, and cluster administration. Record actor, target, operation,
outcome, and deployment revision without recording secret content.

Expose a Prometheus-scrapable endpoint for control-plane health, connected agents,
last successful inventory, request latency/errors, deployment results, queue
state, stream counts, and reconciliation drift. Use bounded labels; never label
metrics with paths, secret values, or unbounded request IDs. Keep full container
and node metrics available through existing exporters rather than rebuilding a
monitoring stack.

Templates produce normal, editable Compose plus declared inputs. Support
versioned local and Git template catalogs, validation, secret inputs, and a
preview of the resulting deployment bundle.

## Delivery gates

1. Complete socket-free control-plane mode: remove implicit local selection,
   local scheduling and readiness dependencies; show disconnected agents clearly.
   Demonstrate global agents on every node, including the manager host.
2. Verify node routing end to end with two workers: exec, logs, file upload,
   edits, task migration, disconnect, and reconnect. Test the real RBD plugin
   with an active consumer and an unattached volume without multi-attachment.
3. Audit and complete native resource discovery and management, immutable
   config/secret replacement, and standard Compose deployment. Existing external
   deployments must appear without onboarding mutations.
4. Implement shared inputs and Harbor rewriting/auth across deployment and
   helper paths. Verify rescheduling onto a node with no cached image.
5. Implement direct/Git deployment bundles, adoption, drift, SOPS export, and
   SwarmCD ownership. Verify a UI edit cannot race another reconciler.
6. Verify OIDC authorization, audit, Prometheus output, templates, backup/restore,
   and upgrade migrations. Run a disposable multi-manager Swarm through failover
   before any production cutover.

The first useful release gate is items 1 and 2, not a feature-count milestone.

## Harbor publishing

The Gitea workflow follows the existing Constructorfleet Harbor publishing
pattern and uses organization-level `DOCKER_REGISTRY_USERNAME` and
`DOCKER_REGISTRY_PASSWORD` secrets. `RELEASE` defines the registry, repository
prefix, version, platforms, and provenance setting. Both upstream Dockerfiles
are built independently, with separate build caches. A missing separator in
the upstream linker flags was corrected in both Dockerfiles so the version
and revision flags remain separate arguments.

Published repositories:

- `docker.home.prettybaked.com/constructorfleet/arcane-manager`
- `docker.home.prettybaked.com/constructorfleet/arcane-agent`

Each receives the release version, `latest`, and `sha-<commit>` tags. Use an
immutable SHA tag or digest when deploying. Image publication does not imply
completion of socket-free control-plane or RBD integration testing.
