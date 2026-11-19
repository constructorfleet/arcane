# 3.0.1-cf.32

## Service replica input validation

- Model replicas as a number or an empty numeric input, matching Svelte’s number binding. Keep numeric initial values and validate them without calling string methods.
- Preserve valid zero replicas, show validation feedback for empty/negative/fractional/unsafe counts, and allow correcting an invalid draft before Save. Refs #84 and #28.
- This follows cf.31 live acceptance of metadata-only editing: the owned Docker-created service retained its original ID, zero replicas, exact Args and complete remaining spec, with no new tasks. Numeric editing exposed the separate validation mismatch fixed here.

## Validation

- Thirty source-executed component checks cover proxy-backed numeric/empty inputs and exact native-field preservation. Existing frontend tests, production frontend build, formatting, full lint and release contract passed.
- Live acceptance uses the retained owned service only: invalid input must show feedback without native mutation, then an edited zero and a label must save while preserving the exact remaining native spec. No production service, node role, port or RBD mount is changed.
