# 3.0.1-cf.42

- Keep image polling, auto-update, auto-patch, auto-heal, scheduled prune, stale volume helper cleanup and local volume backup job registration on Docker-capable node agents or legacy managers (#102, CP-02).
- Prevent settings changes from registering those Docker-only jobs on the control plane. Preserve remote-agent health checks, GitOps coordination, project/template file watching and database/file maintenance.
- Reject control-plane Docker client acquisition and refresh with a classified conflict before probing any daemon, including bound client copies. Node-agent and legacy client behavior is preserved.

Existing Docker client, scheduler settings and dependency-wiring tests cover the mode boundary. Full release CI and deployed runtime verification are recorded in the PR/issue. This does not implement socket-free system backup execution: manager backup coordination and repository retention remain registered, and Docker-dependent execution fails at the client boundary. Backup support and broader CP-02 acceptance remain follow-ups.
