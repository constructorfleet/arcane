# 3.0.1-cf.100

- Count running workloads in the Images repository summary. On Swarm managers, require both desired and current task states to be Running. Stopped tasks and idle caches remain visible in the node selector.
- Count copies pinned to the same digest as a single variant across nodes.
