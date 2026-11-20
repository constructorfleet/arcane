# 3.0.1-cf.59

- Preserve exact untouched native healthcheck timings when editing an existing custom healthcheck (#138 / #28 / #125). The container form sends only changed whole-second settings, retries or Test through a purpose-specific partial update contract.
- Keep the existing full-replacement/create API and inherit/disable/new-custom paths. Partial updates reject replacement/reset conflicts, invalid values and missing/disabled custom targets; omitted fields and Test slices are preserved without mutating inspected input.

This fixes preservation during another field’s edit; it does not add fractional-duration authoring or exact fractional display. Native Docker constraints remain authoritative. Source/isolated checks do not establish live recreation, scheduler execution or cleanup acceptance.
