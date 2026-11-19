# 3.0.1-cf.26

## Fractional container CPU limits

- Allow fractional CPU values in the shared standalone-container Create/Edit form. Forward the numeric step to the actual input and use an unrestricted fractional step only for CPUs. Refs #69.
- Preserve existing nonnegative validation and NanoCPU conversion; other numeric inputs keep their defaults.

## Validation

- The original cf.23 form rejected 0.1 with browser step validation and accepted 1; native Compose deployment independently confirmed that 0.1 is supported by Docker.
- Existing frontend tests, production build, formatting, full lint and release contract pass locally. Runtime fractional Create/Edit acceptance follows normal publication and rollout.
