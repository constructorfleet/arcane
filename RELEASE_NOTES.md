# 3.0.1-cf.75

- Check the container's current state on its agent before counting a recent exit or OOM event under Home's Needs Attention. Recovered or removed containers clear their warning. (#182)
- Refresh active attention inputs while Home is open. Failed containers that are still exited, dead, or restarting remain eligible, and the Events page retains the history.
- Group repeated failed Swarm tasks by service slot. Clear the warning when a healthy replacement is observed for that slot.
