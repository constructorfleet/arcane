# 3.0.1-cf.103

- Read Harbor vulnerability reports for exact image digests through the saved registry connection and compare a candidate on the selected platform.
- Resolve Go lint findings in Harbor report loading, discovery, and test fixtures.
- Native QA against `constructorfleet/nginx-utility:1.30.6` verified amd64/arm64 index resolution and exact digest report parity: each platform returned 11 findings (7 Medium, 4 Low). Same-report comparison returned 11 persisting, zero added, and zero removed; unavailable platforms and unscanned artifacts returned their expected states.
- Native comparison QA against `constructorfleet/compass` indexes `sha256:1815a1e9563c09c5b0118c6a14033a717ebb2f03f7c8c2643d0d3f3b78928667` and `sha256:d53ac4c8aeb49e2bc13cf69113d19ca142b9f7ce56c149db351ef3db32074762` passed on amd64 and arm64. Each report contained 14 findings (3 High, 7 Medium, 4 Low); the comparison returned 14 persisting, zero added, and zero removed in both directions. The indexes share their amd64 child; arm64 resolves to distinct children. Service results matched direct child-digest requests.
- QA limitation: browser interaction and deployed saved credentials were not verified; the live preview was left untouched.
- Keep autologin opt-in within its owning tests instead of enabling it globally in the backend test recipe, preserving forced-password-change coverage.
- Verify exact project workspace directory entries so duplicate-creation checks work on both case-sensitive and case-insensitive filesystems.
