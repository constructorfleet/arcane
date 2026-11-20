# 3.0.1-cf.41

- Explain standard Docker/OCI registry host and namespace URLs, explicit docker.io, and password/access-token/robot-secret credentials (#99).
- Clarify optional known repository names, replace the misleading /api example with project/image, and label the manual repository input. Explain that repository access can work when global catalog listing is denied or unsupported.
- Explain that registry browsing uses the remote registry while image Pull/Push use the selected agent. Clarify private Swarm image credential forwarding with the label Send registry credentials to Swarm and an adjacent help text.

Credential storage, authorization, repository access and deployment behavior are unchanged. Swarm credential forwarding remains opt-in and false by default. Production frontend build, required formatting/lint and exact-head CI are recorded in the fixing PR; post-release UI checks inspect the guidance without submitting credentials or changing registry permissions. Provider-neutral runtime acceptance remains separately tracked in #40.
