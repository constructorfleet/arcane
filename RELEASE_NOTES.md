# 3.0.1-cf.102

- Restore the System Diagnostics Goroutines tab in control-plane deployments. It reads the manager's Go runtime directly, including when no Docker agents are registered.
- HTTP checks passed in both manager modes and confirmed requests without diagnostics access are denied. The tab awaits verification on the deployed release.
