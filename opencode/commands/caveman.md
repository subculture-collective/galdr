---
description: Set caveman mode, default, or status
---
Interpret `$ARGUMENTS` as a caveman control command and use the `caveman_mode` tool.

Rules:

- no args => action `set`, mode `ultra`
- `lite|full|ultra|wenyan-lite|wenyan|wenyan-ultra|commit|review` => action `set`
- `off|normal|stop` => action `set`, mode `normal`
- `status` => action `get`
- `clear` => action `clear`
- `default <mode>` => action `set-default`
- `default reset` => action `reset-default`
- `list` or `help` => action `list`

After the tool call, reply in one short line with the active session mode and default.
