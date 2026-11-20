# 3.0.1-cf.50

- Keep whitespace-only env and override source files visible when reopening Swarm stack editors, and env files visible in project creation and the stack Source pane (#118).
- Use raw nonempty source length for file presence. Semantic validation, source bytes, exact-empty/absent behavior and Git-derived environment handling are unchanged.

Source review and frontend checks cover the visibility change. Live acceptance will cold-reopen the retained owned zero-task cf.47 fixture, whose exact source bytes already persist, then verify guarded cleanup. No deployed application stack, global template, volume or plugin operation is changed.
