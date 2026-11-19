# 3.0.1-cf.19

## Fixes

- Read published Constructorfleet releases, including prereleases, from Gitea. Select the newest semantic version across release pages, exclude drafts and invalid tags, and use fork release/changelog links (#43).
- Report deployment-managed updates for socket-free control planes and detected Swarm tasks. Link the running release while leaving update checks and installation to the deployment owner.
- Refuse direct Swarm-task recreation and control-plane upgrade jobs before image pulls or deployment mutations. SwarmCD continues to own this homelab deployment.
- Preserve the current image repository when resolving standalone upgrades; no registry migration or updater action is performed by this change.

## Validation and limits

- Validate release selection, pagination, cancellation, deployment policy and existing upgrade image-selection tests, frontend checks/build, format/lint and required CI before merge.
- Runtime release-link and action visibility acceptance follows publication. No production in-app upgrade is used as a QA test. This does not implement general GitOps ownership detection for arbitrary standalone containers.

- The configured Gitea repository is private: unauthenticated standalone release polling is unavailable. Private-feed credentials are not provisioned or copied by this change; #43 remains open for that broader follow-up. Deployment-managed instances do not poll the feed.
