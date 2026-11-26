# 3.0.1-cf.106

- Use Harbor as the vulnerability source for running image digests, including exact platform resolution and configured Harbor proxy-cache projects. Scan submits to Harbor and polls its report; saved credential and provider errors remain visible.
- Retain local Trivy scanning when the image cannot be obtained or scanned through Harbor, including local builds without registry digests. Manual and scheduled targets exclude cached images without running consumers.
- Show one image vulnerability tab and deduplicate running replicas by image identity. Remove the separate registry-report tab and image host/version scan selectors. Security views import Harbor findings, CVSS scores, descriptions, and completed report times; risk exposure aggregation continues to represent node consumers.
- Validation: backend race/coverage suite, focused registry/vulnerability/image tests, frontend tests, production frontend build, formatting and lint. Native Harbor reads verified Compass amd64 (14 findings), Redis through the Docker Hub proxy (239 findings), and unavailable-artifact/local-build fallback selection.
- QA #252 remains open: authenticated Harbor scan submission, local fallback execution, and interactive swarm acceptance require runtime confirmation. The older Compass index is now unavailable, so distinct-artifact comparison needs a fresh pair.
