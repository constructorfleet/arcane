# 3.0.1-cf.34

- Stop stack deployment when its Compose, environment or supporting source files cannot be persisted (#88, CMP-07). Return the storage error before any Docker deployment request instead of reporting a managed stack without saved source.
- Extend existing stack-source business coverage with an owned temporary storage failure and fake Docker endpoint. Before the fix, network/service creation and a success response continued; after the fix, the source error is returned with no Docker requests.
- Related stack-source and native Swarm library coverage passes. Full checks and runtime release acceptance are recorded in the fixing PR and issue.

The failure case uses local temporary files and an in-process fake Docker endpoint. No production source directory, stack or service is changed to induce a failure. Broader Compose/GitOps design remains tracked separately.
