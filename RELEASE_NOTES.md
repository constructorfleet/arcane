# 3.0.1-cf.53

- Preserve untouched standalone-container command, entrypoint and healthcheck settings during edits by omitting them from the recreate request (#125). Keep empty arguments, native exec healthchecks and unexposed healthcheck options intact.
- Changing healthcheck timing retains an unchanged exec Test vector and existing StartInterval. Explicit mode/test edits and resets keep their current behavior.

Validation exercises the actual owning form against native-array, healthcheck, timing and reset cases; live acceptance will use an owned disposable container. No production container is edited or recreated. Intentional standalone argument authoring still uses the existing text syntax; broader #28 remains open.
