# 3.0.1-cf.56

- Preserve native network attachments when a standalone-container edit leaves the network rows untouched (#130). Omit the frontend network replacement and retain explicit IPv4/IPv6/link-local IPAM through the shared backend recreation path used by edit, redeploy and rollback.
- Keep the existing Docker API compatibility sanitization and clone native endpoint settings rather than aliasing inspected data. Filter the previous container’s generated short-ID alias so the daemon can regenerate it.

Validation covers actual owning form payloads and the existing backend network-edit test. This bounded fix preserves untouched networking; intentional network-row edits retain the existing replacement contract. Full IPv6/link-local authoring and reset controls remain follow-up work under #28. No production network is changed for QA.
