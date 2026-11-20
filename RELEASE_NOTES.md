# 3.0.1-cf.52

- Edit Docker service Command and Args as individual literal arguments rather than splitting a flat string on spaces (#123). Preserve spaces, quotes, newlines and intentionally empty entries.
- Removing every row restores the image default for that field; untouched arrays and other native service fields retain their existing preservation behavior.

Validation exercises the actual service-editor form and submitted native spec with boundary cases and metadata-only edits. Live acceptance will use only an owned zero-task service; no production command or workload is changed. Broader Compose/GUI fidelity remains #28.
