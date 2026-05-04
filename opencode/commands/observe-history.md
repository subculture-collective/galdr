---
description: Show recent observe loop history
---
Interpret `$ARGUMENTS` as `[loop-id] [limit]`.

Rules:

- if first token is numeric, treat it as `limit`
- if second token is numeric, treat it as `limit`
- otherwise first token = `loop_id`

Use the `observe_history` tool.

Reply with newest useful events first.
