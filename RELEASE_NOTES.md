# 3.0.1-cf.73

- Replace the repeated Nodes table on Cluster with an overview grouped by native Swarm ID. See the leader, manager reachability and quorum requirement, scheduling state, agent coverage, service replica shortfalls, and reclaimable Docker space with reporting coverage. Existing Cluster administration is still available. Reclaimable estimates cover images, build cache, and stopped containers. (#175)
- Open Node Details from the hostname or Inspect while keeping View Tasks separate. The page shows Swarm identity, labels and status, Docker-reported host details and drivers, the associated agent, and scoped resource links. Node-local reads verify that Docker reports the requested Swarm node ID. (#174)
- This first release covers read-only inspection. Existing Swarm controls remain on Nodes. Dashboard layout changes, node cleanup, and host operation design are tracked separately in the redesign. (#177)
