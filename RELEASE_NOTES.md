# 3.0.1-cf.51

- Expose system recovery mode/provider support and reject unsupported new backup admission before leases, key adoption, run records or workflow submission (#120). Preserve authorization before rejection. Whole-restore service admission rejects before leases/helper work; its existing handler activity lifecycle remains.
- Reconcile unsupported system schedules without changing stored policy metadata, and guide new UI jobs to agent volume backups. Untouched enabled rows survive collection updates so individual existing policies can still be disabled; missing capability metadata is treated as unknown.

This is admission and UI guidance, not control-plane backup execution. #104 remains open for actual execution/delegation; existing persisted recovery/resume and remote-agent volume paths retain their safeguards. Validation uses service-level policy/admission checks and read-only UI capability inspection, without running backup, restore, helper, repository or storage operations.
