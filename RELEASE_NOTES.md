# 3.0.1-cf.109

- Remove the global volume list's misleading Swarm global volume expansion and the detail title's workspace attachment subtext. Preserve existing logical volume identity and owning-agent workspace routing.
- Prefer the driver's creation-time size option in the Size column, including on mobile. Show a dash for plugin volumes without that option; retain Docker's existing size reporting for local volumes.
- Rename the list's Backing filesystem column to Filesystem disk usage and show compact used / total values, or a dash when unavailable. Keep workspace access in actions and the detail page. Usage probes read existing helpers without creating a mount.
- Order the overview as Volume details, Driver options, Labels, Active mount information, then Containers using this volume. Include filesystem usage in Volume details only when available. Active mounts show the agent and mountpoint for running application or helper consumers; empty and stopped-only agent records are excluded.
- Validation: required formatting and lint, frontend production build and unit tests, and owning volume/workspace Go tests on the combined cf.108 main and volume branch.
- QA #260 (VOL-03 and VOL-09) remains open for authenticated browser acceptance and deployed-release verification. The dev stack was not started or rebuilt. No live RBD failover or attachment behavior is claimed.
