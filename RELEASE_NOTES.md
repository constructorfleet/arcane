# 3.0.1-cf.4

## Changes

- Require feature branches and squash pull requests for changes to main.
- Require Go tests and frontend build/type/lint checks before merge.
- Validate the release version and notes before merging; build manager and agent images before promoting latest.
- Publish a Gitea release with the source commit, merged PR, Harbor artifact links, version tags, and immutable image digests.
- Add QA issue intake and triage conventions linking defects, fixes, checklist items, and release notes.
- Record verified cf.3 volume-list recovery in the feature checklist.

## Validation and limitations

- Release tooling has local automated coverage; the first merged release validates the new Gitea pipeline end to end.
- Main branch protection and required checks are configured in Gitea; repository instructions describe their maintenance.
- SwarmCD owns runtime rollout. A successful image build is not runtime feature acceptance.
- Harbor cannot atomically move two repository tags: if promotion fails partway, retry the same workflow and verify both digests.
