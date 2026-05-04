# Sandcastle OpenCode Service Integration Plan

## Goal

Replace Sandcastle's current `opencode run ...` CLI usage with a custom Sandcastle provider that calls a lazily-started OpenCode HTTP service inside each Docker sandbox.

Core invariant: each Sandcastle Docker sandbox owns exactly one OpenCode service on container-local `127.0.0.1:4096`. No shared host/global OpenCode service.

## Locked design decisions

- One OpenCode service per Sandcastle Docker sandbox/container/worktree.
- Lazy service startup inside the custom agent command, not `onSandboxReady`.
- Use a repo-owned Node wrapper script as the integration seam.
- Use raw HTTP `fetch`, not `@opencode-ai/sdk`.
- Use Basic Auth when `OPENCODE_SERVER_PASSWORD` exists; username defaults to `opencode`.
- Support exact per-role OpenCode model IDs:
  - `SANDCASTLE_OPENCODE_PLANNER_MODEL`
  - `SANDCASTLE_OPENCODE_IMPLEMENTER_MODEL`
  - `SANDCASTLE_OPENCODE_REVIEWER_MODEL`
  - `SANDCASTLE_OPENCODE_MERGER_MODEL`
- If role model env is absent, omit `model` and use OpenCode service defaults.
- Pass prompt through provider `stdin`; wrapper reads stdin.
- Fresh OpenCode session per Sandcastle agent run.
- Reuse one OpenCode service process per sandbox container.
- Store PID/log/port state outside repo under `/tmp/sandcastle-opencode/`.
- Use fixed internal port `4096` in each container.
- Use blocking `POST /session/:id/message`, not async prompt endpoints.
- stdout is final assistant text only; service/debug logs go to stderr.
- Do not pass OpenCode `agent`, `tools`, `system`, or `noReply` by default.

## Files to change

### 1. `.sandcastle/main.mts`

Add inline local provider factory:

```ts
function opencodeService(opts: {
  role: "planner" | "implementer" | "reviewer" | "merger";
  title?: string;
}): sandcastle.AgentProvider
```

Provider behavior:

- `name`: `"opencode-service"`
- `captureSessions`: `false`
- `env`: `{}`
- `buildPrintCommand({ prompt })`: returns a command running `.sandcastle/opencode-service-agent.mts` with `--role` and optional `--title`, plus `stdin: prompt`
- `parseStreamLine()`: returns `[]`

Replace the four existing `sandcastle.opencode("opencode/big-pickle")` calls with:

- planner: `opencodeService({ role: "planner", title: "planner" })`
- implementer: `opencodeService({ role: "implementer", title: issue.title })`
- reviewer: `opencodeService({ role: "reviewer", title: issue.title })`
- merger: `opencodeService({ role: "merger", title: "merger" })`

Do not alter prompts, iteration counts, hooks, Docker config, or workflow.

### 2. `.sandcastle/opencode-service-agent.mts`

Responsibilities:

1. Parse CLI args: `--role planner|implementer|reviewer|merger`, `--title <text>`, `--healthcheck-only`.
2. Read full prompt from stdin.
3. Validate runtime: `opencode` exists, `fetch` exists, role valid.
4. Fixed service config:
   - base URL: `http://127.0.0.1:4096`
   - state dir: `/tmp/sandcastle-opencode`
   - PID file: `/tmp/sandcastle-opencode/opencode.pid`
   - log file: `/tmp/sandcastle-opencode/opencode.log`
5. Auth: if `OPENCODE_SERVER_PASSWORD` exists, send Basic Auth; username from `OPENCODE_SERVER_USERNAME ?? "opencode"`.
6. Service lifecycle:
   - check `GET /global/health`
   - if healthy, reuse
   - if unhealthy and PID exists, kill stale PID once
   - start `opencode serve --hostname 127.0.0.1 --port 4096` in background
   - redirect logs to `/tmp/sandcastle-opencode/opencode.log`
   - wait up to 30s for health
   - on failure print log tail to stderr and exit nonzero
7. `--healthcheck-only`: validate + ensure service healthy; exit 0; no session creation.
8. Fresh session per run: `POST /session` with body `{ "title": "sandcastle <role>: <title>" }`.
9. Blocking prompt call: `POST /session/:id/message` with body `{ "parts": [{ "type": "text", "text": "<stdin prompt>" }] }`; add `model` only if matching role env exists.
10. Output policy: stdout final assistant text only; stderr health/start/debug/log tail; compact JSON fallback to stdout if no text parts.

### 3. `.sandcastle/.env.example`

Add docs:

```env
# OpenCode service auth
OPENCODE_SERVER_USERNAME=opencode
OPENCODE_SERVER_PASSWORD=

# Optional exact OpenCode model IDs per Sandcastle role.
# Leave blank to use OpenCode service defaults.
SANDCASTLE_OPENCODE_PLANNER_MODEL=
SANDCASTLE_OPENCODE_IMPLEMENTER_MODEL=
SANDCASTLE_OPENCODE_REVIEWER_MODEL=
SANDCASTLE_OPENCODE_MERGER_MODEL=
```

Keep `OPENCODE_API_KEY` only as legacy/provider-specific note if needed by OpenCode itself. Sandcastle wrapper must not require it.

## Verification plan

1. Wrapper CLI/syntax smoke:
   ```bash
   npx tsx .sandcastle/opencode-service-agent.mts --help
   ```
2. Real sandbox health smoke:
   ```bash
   npx tsx .sandcastle/opencode-service-agent.mts --role planner --healthcheck-only
   ```
   Expected: service starts at container-local `127.0.0.1:4096`, health passes, no dirty worktree, logs only under `/tmp/sandcastle-opencode`.
3. Minimal prompt test inside sandbox:
   ```bash
   printf "Reply with OK only." | npx tsx .sandcastle/opencode-service-agent.mts --role planner --title smoke
   ```
   Expected stdout: `OK`; stderr may contain service lifecycle logs.
4. Full Sandcastle smoke:
   ```bash
   npm run sandcastle
   ```
   Validate planner emits parseable `<plan>...</plan>`, implementer/reviewer/merger route through wrapper, each Docker sandbox gets its own service, no host/global OpenCode service is used, `.sandcastle/worktrees` behavior unchanged.

## Acceptance criteria

- No new npm dependencies.
- `.sandcastle/Dockerfile` unchanged.
- No prompt/orchestration behavior changes except OpenCode transport.
- Each sandbox container lazily starts/reuses its own OpenCode service.
- Fixed internal port `4096`.
- Fresh OpenCode session per Sandcastle agent run.
- Per-role model override supported via exact OpenCode model IDs.
- Missing role model env means use OpenCode default.
- stdout remains clean agent output.
- logs/debug stay stderr or `/tmp/sandcastle-opencode/opencode.log`.
- wrapper healthcheck works inside Docker sandbox.
