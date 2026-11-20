# 3.0.1-cf.62

- Compare normalized Docker image references when a Swarm stack source is saved. Equivalent Docker Hub full and familiar names no longer request image resolution or discard an existing resolved digest during source-only edits. Preserve its resolved platform metadata while applying the desired placement constraints and preferences.
- Honor explicitly changed digest pins, including private registries, while treating equivalent repository/digest references alike regardless of decorative tags, instead of treating every digest for the same repository/tag as unchanged.

Validation: reproduced the cf.61 source-only save changing unrelated platform metadata on an owned zero-replica fixture. Isolated checks exercise the actual owning comparison/preservation functions across equivalent names, default tags, resolved tags, explicit pins and real repository/tag/registry changes. Existing Go, formatting, lint and release checks run before merge; live post-release revision and no-op acceptance remain required. No running production workload or real secret value was used for the reproduction.
