# 3.0.1-cf.38

- Apply Diagnostics Pause to the independent Actors and backend-log views (#93). Resume immediately displays the newest actor snapshot and the bounded log history received while paused.
- Keep the existing log-buffer limit, filters and explicit Clear behavior. Paused reconnects preserve the frozen view, and automatic scrolling follows displayed updates.
- Extend the existing duplicate-log browser case with pause/resume buffering. A network-free harness exercises actual component callbacks/effects and the Clear expression: five failures before the correction, all six cases passing afterward.

Live cf.34 Actors reproduced the failure while the parent header stayed frozen; no actor details, profiling or jobs were triggered. The harness mocks framework scheduling and does not establish full DOM/browser acceptance. Full development E2E was not run; no dev stack was started. Required checks and post-release live acceptance are recorded in the PR and #93. Broader administration and scoped-authorization acceptance remain #16/#63.
