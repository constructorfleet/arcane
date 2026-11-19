# 3.0.1-cf.37

- Preserve the inactive-volume attachment error through the workspace API (#14). Use its workspace-specific conflict category so the UI receives HTTP 409 and the driver/agent guidance, instead of a generic 500 “internal error”.
- Strengthen the existing inactive cached-helper and fresh-create business cases to assert the category consumed by the workspace handler. Both cases failed on cf.36 despite matching the broader conflict category.

Live cf.36 returned promptly on the user-supplied inactive swarm03 URL, but lost the error details at the workspace boundary; #14 remains open for corrected live acceptance. The preflight still precedes helper acquisition. No inactive RBD mount or file operation is used to reproduce this classification defect. Normal volume/workspace/backup coverage, required checks and post-release UI/no-helper audit are recorded in the PR and issue. Broader shared identity, active-agent navigation and RBD writes remain #13/#33.
