# 3.0.1-cf.87

- Run ephemeral Trivy containers on the default bridge when the automatically detected Swarm network disallows standalone attachments. (#201)
- Preserve attachable and local networks and explicit Trivy network settings. Verify the Docker network cases in the scanner tests. (#201)
