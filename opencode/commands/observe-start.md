---
description: Start bounded observation loop for current session
---
Interpret `$ARGUMENTS` as: `<target> [interval-seconds] [success criteria...]`.

Use the `observe_start` tool.

Rules:

- first token = target
- second token, if numeric = `interval_secs`
- remaining text = `success_criteria`
- if interval missing, let tool default

Reply with one short line containing loop ID, target, interval, and next check time if shown.
