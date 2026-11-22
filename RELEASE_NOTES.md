# 3.0.1-cf.89

- Apply the existing URL secret redactor to query strings and referrers in debug request logs. (#169)
- Verify URL-encoded query values are redacted while unrelated parameters remain available for diagnostics. (#169)
