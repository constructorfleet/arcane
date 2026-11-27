# 3.0.1-cf.111

- Version the approved resource contract, complete twelve-resource column inventory, relationship mappings, interactive wireframe and source-inspected implementation matrix. Track missing scoped views, task policy, image/security architecture and separate manual stack source import/export explicitly.
- Apply the approved no-labels-in-tables rule to Swarm secrets/configs, Podman secrets, containers and projects. Remove optional/hidden label columns and the mobile secret/config label summary while retaining labels in resource data/details and existing filter behavior.
- Remove Pin this view from container/image/network/volume/project lists and resource pins from agent/container/project row menus. Keep dashboard widget configuration and existing detail-page pin actions for the later detail/action alignment batch.
- Remove Release notes and Commit from container list actions and their now-unused list dialogs/state. Preserve Convert to Compose, container lifecycle actions, and image release notes/scanning actions.
- Remove the unused list-pinning helper. This is the first application cleanup batch under UX-02; shared menu grouping, title/Overview updates, task policies and missing relationship views remain tracked in RESOURCE_IMPLEMENTATION.md.
- Validation: 25 frontend unit tests, production build and full formatting/lint passed locally; required checks are rerun on the final diff. No backend behavior changed and no dev stack was started/rebuilt. Authenticated UI acceptance follows the normal QA rollout.
