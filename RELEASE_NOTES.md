# 3.0.1-cf.58

- Round-trip and edit static IPv6 and link-local addresses alongside IPv4 and aliases in standalone create/edit network rows (#136). Alias-only changes retain all represented IPAM values instead of dropping hidden fields.
- Share complete address parsing between create/edit, reject wrong address families, scoped IPv6/link-local entries and malformed addresses before container creation or recreation, and retain omission for untouched network rows.

Blank static fields return address allocation to Docker; a blank link-local list clears that configured list. This does not claim detach-all from removing network rows, disable IPv6 on the network, or implement network connectivity testing. Validation/live acceptance uses owned isolated resources; broader #28/#13 remains open.
