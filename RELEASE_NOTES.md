# 3.0.1-cf.10

## Fixes

- Bound volume workspace folder/text reads to 30 seconds and propagate query cancellation to HTTP requests (#14).
- Disable automatic folder-read retries and background focus/reconnect retries. Offer explicit Retry and return-to-Volumes actions with the server error and attachment guidance.
- Add Stop waiting during initial loading to cancel the read and return to the overview. Upload/save/download behavior and storage attachment policy are unchanged.

## Validation and limits

- Validate frontend build/type/lint and required CI before merge.
- This bounds the interactive wait; it does not prove the cause of the reported inactive-RBD failure, relocate volumes, or change exclusive mapping. Shared-volume grouping/active-node navigation remains #13; exact failing-volume acceptance remains separate.
