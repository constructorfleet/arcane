# 3.0.1-cf.77

- Add a Force update quick action to Swarm service rows and Service Details. Docker recreates tasks using the current service specification and update policy. (#185)
- Offer Force update service from an active task row, including tasks stuck in Preparing. The confirmation identifies the affected service and explains the impact on its tasks. (#185)
- Open existing Scale and Edit forms from service row menus, and offer View service and Edit service on task rows. (#185)
