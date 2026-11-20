# 3.0.1-cf.57

- Show separate matching-name/driver records on other permitted agents for global-driver volumes (#132 / #13), with reported mountpoints and container references. Explicit links open the chosen agent's Overview.
- Label shared-storage identity as unverified; expose failed or excluded metadata reads. Discovery requires volume-list access, destination links require volume-read access, and inactive Workspace guidance offers a route back to these observations.

- Require active Workspace intent for folder/file reads, reset that intent on Overview, and start a fresh detail view when changing agents (#134). Confirm discarding pending drafts on ordinary agent-link clicks; modified/new-tab navigation keeps the current draft. Already-issued file reads are not newly promised cancellation.

This is metadata discovery/navigation, not shared-backend deduplication or automatic attachment routing. It creates no workspace helper and does not mount an inactive RBD volume. Application/helper/unknown consumer classification remains on each destination's Overview; reference counts can include helpers/stopped containers and are not proof of active application use. Broader #13 remains open.
