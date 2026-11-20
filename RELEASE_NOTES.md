# 3.0.1-cf.45

- Check global-volume Swarm service references before delete or rename can stop helpers or issue mutations (#109).
- Protect current and rollback service volume mounts even on zero-replica services, plus nonterminal/orphaned task references. Force does not bypass this dependency guard; unknown membership/reference state fails closed with a conflict or unavailable error.
- Keep local-volume and standalone-host admission unchanged when metadata is readable. Missing/unreadable inspection fails before mutation even with Force, so stale daemon records cannot be force-cleared through an unverifiable operation. A Swarm worker must use a manager agent to verify global-volume references; automatic worker-to-manager safety coordination remains broader work.

Existing volume service coverage checks current/rollback/task references, worker/locked/omitted state, unavailable metadata and unreferenced cases with a fake Docker API and asserts no helper or mutation calls before rejection. Production validation remains read-only: no real-volume delete, rename, mapping or plugin operation is used to exercise the guard. This is a conservative Swarm dependency check, not proof of shared backend identity or an atomic cross-system transaction. Shared-volume grouping remains #13; broader plugin operations remain #33.
