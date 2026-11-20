# 3.0.1-cf.60

- Preserve SCTP port mappings during standalone container edits instead of silently changing them to TCP (#139 / #28). Mixed TCP, UDP and SCTP bindings on the same numeric port stay distinct.
- Offer SCTP alongside TCP and UDP in the shared port editor used by container and pod forms; retain host IP/port values and explicit protocol changes or removal.

This fixes form/API fidelity. It does not claim live SCTP packet flow, listeners or host-kernel support. Runtime verification uses owned resources or canceled drafts only; no production port mapping is changed for QA.
