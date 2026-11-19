# 3.0.1-cf.33

- Preserve an existing volume file's UID/GID and mode when publishing an atomic workspace edit; refuse publication if ownership or mode cannot be restored (#86, VOL-05/VOL-07).
- Package the corrected Constructorfleet ACFS companion in both release images. Selected agents copy it into each helper's private filesystem; existing helper utilities, mount scope and execution identity remain intact.
- Require an advertised ownership-preservation capability before updating existing files. Legacy helper reads remain available when a companion is absent; unsafe updates fail before staging or backup mutation.
- Validate the dependency's Linux suite, ownership behavior, helper installation and existing volume coverage before publication. Runtime acceptance remains pending the normal SwarmCD rollout and owned local-volume UI save.

No production or RBD file is used as a test fixture. Independent RBD lifecycle and scoped-login QA remain open.
