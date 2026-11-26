# 3.0.1-cf.104

- Make image detail actions and tabs available from list menus, including Push, release notes, history, attestations, and registry vulnerability reports. Grouped images expose operations for each explicitly identified owning agent and image copy.
- Add missing detail navigation to container, project, volume, network, agent, Swarm, Podman, and template lists. Add container commit and dashboard pins, project and agent dashboard pins, agent sync, and Swarm service rollback access.
- Keep selection and action columns visible despite legacy saved column preferences. Volume selection retains metadata across pages and bulk removal excludes in-use, logical, protected backup, and unauthorized volumes.
- Validate image copy metadata and operation targets, table column preferences, frontend type/lint checks, and backend tests before publication.
- Runtime limitation: interactive menu and destructive action acceptance on the live Swarm require human QA; automated checks do not establish that acceptance.
