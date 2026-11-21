# 3.0.1-cf.76

- Consolidate the Containers, Images, Volumes, and Networks shortcuts under Local Resources in Node Details. Each opens on the verified node agent. (#174)
- Add paginated Docker network inventory with driver, scope, subnet, gateway, and IP range, and link each readable network to its details. (#174)
- Show installed Docker-managed plugins with enabled state, source reference, and capability, alongside effective daemon settings reported by Docker. Reading daemon.json requires host access. (#174)
