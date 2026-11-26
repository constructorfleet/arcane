# 3.0.1-cf.107

- Keep image inventory, dashboard summaries, vulnerability lists, and risk reads independent of Harbor collection. These paths read stored report summaries and findings rather than fetching and rewriting every image report during a page request.
- Resolve proxy-cache reports by the exact platform manifest digest instead of the upstream multi-platform index digest. Treat Harbor background indexing as not scanned and retry it within explicit scan work, without failing inventory pages.
- Skip persistence when a report has the same source, state, and completed scan timestamp, avoiding repeated finding deletion and insertion from unchanged detail reads.
- Validation: owning registry, vulnerability, and image race tests; fixture coverage for proxy indexes whose original digest has no artifact record, delayed child indexing, and stored summaries with an unavailable registry; required formatting and lint.
- QA #254 remains open for authenticated live Images/Security/dashboard acceptance and browser stability. The wider fleet-scale inventory and registry synchronization architecture still needs explicit load validation; this release does not claim it.
