# 3.0.1-cf.7

## Fixes

- Label current and desired state side by side on service task cards, including historical tasks (#6).
- Populate task errors only from Docker's error field; ordinary status messages such as started and shutdown no longer appear as errors in service, stack, or cluster task views (#7).

## Validation

- Check healthy running tasks, normal shutdown history, rejected/failed tasks, and actual Docker errors through the shared task-summary conversion.
- Required release-contract, Go, and frontend checks run before merge.
- Node/container navigation and node-agent coverage remain separate open QA items (#5, #8).
