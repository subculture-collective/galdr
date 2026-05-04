---
description: Compress a prose-heavy file into caveman format
---
Compress the file at `$ARGUMENTS` into caveman format.

Requirements:

- If `$ARGUMENTS` is empty, ask for a file path.
- Only compress prose-heavy natural language files.
- Preserve headings, code blocks, inline code, URLs, commands, file paths, env vars, dates, version numbers, and numeric values exactly.
- Create a backup beside the file named `<original-stem>.original.md` before overwriting.
- If the backup already exists, stop and ask before changing anything.
- Overwrite the original only after the compressed content is ready.
- If the target is code/config instead of prose, refuse and explain briefly.

Use clear prose for any risk or ambiguity. Otherwise stay terse.
