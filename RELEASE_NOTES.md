# 3.0.1-cf.13

## Fixes

- Open the exact failed activity from repeated-failure dashboard attention items, including image update checks, rather than sending those failures to the Updates list (#10).
- Linked job failures also open their matched activity; jobs without a captured activity retain their scoped job-history destination.
- Show the latest activity message when no error summary exists and allow reasons to wrap without desktop truncation. Preserve affected-agent labels and other warning destinations.

## Validation and limits

- Reported Updates misrouting was reproduced on the live dashboard before this change.
- Validate frontend build/type/lint, existing frontend tests, and required CI before merge; verify the activity panel and agent scope after rollout.
