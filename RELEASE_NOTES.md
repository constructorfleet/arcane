# 3.0.1-cf.36

- Inspect the selected agent's volume metadata before acquiring a workspace helper (#14). A global plugin volume with an empty driver-reported mountpoint returns a clear conflict directing the user to an agent already using it, before helper creation or cached-helper reuse.
- Keep unused local volumes accessible and permit global plugin volumes that report an existing local mountpoint. Extend existing helper coverage for inactive fresh/cached helpers, attached plugin helpers and unused local volumes.
- Record the preceding cf.34 local-driver lifecycle acceptance: stale-save rejection, cold reload, exact download bytes, upload, rename and move; original fixture preservation and guarded cleanup were independently checked.

Volume/workspace/backup tests and required checks are recorded in the PR. The reported RBD attachment state was inspected read-only; no inactive volume was mounted to reproduce the problem. The guard does not infer shared identity from a name or global scope, choose another agent automatically, or complete broader plugin/RBD write acceptance. Runtime acceptance is tracked in #14.
