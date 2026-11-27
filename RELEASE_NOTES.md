# 3.0.1-cf.110

- Rebrand the displayed product as HiveMind across the app, CLI, documentation and release metadata. Apply the HiveMind logo, Barnabee wordmark, app icons, favicon, PWA manifest and honey-yellow default accent. The OIDC loading screen respects reduced motion.
- Remove the development-branding artwork and toggle so fork prereleases display the HiveMind brand. Preserve custom uploaded branding through the existing logo slots.
- Preserve repository, image, deployed service, environment-variable, header, import-path and CLI binary names for the upgrade path.
- Reconcile the rebrand with cf.109, preserving running-artifact image identity, Harbor-backed stored-report reads, replicated mixed-image warnings and the volume overview/capacity cleanup.
- Validation: full formatting and lint, backend and CLI race/coverage suites, 25 frontend unit tests and the production frontend build passed on the integrated branch. Authenticated browser acceptance will follow the normal main release and SwarmCD rollout; no dev stack was started or rebuilt.
