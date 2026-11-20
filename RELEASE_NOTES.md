# 3.0.1-cf.71

- Show the selected Swarm service's tasks in a searchable table with pagination, sorting, node and state filters, and accurate totals (#171). The table includes current and desired state, node, container access, update time, and errors.
- Show clearly separated vertical cards on mobile. Task container links still resolve through the agent on the node running the task.
- Validate pagination and combined filtering with a 120-task data set. Deployed UI verification follows publication.
