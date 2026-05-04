---
name: parallel-agents
description: "Run multiple Claude Code instances in parallel using git worktrees, branch isolation, and the Claude Code SDK. Covers the master-clone model, parallel scripting with claude -p, and multi-agent coordination patterns. Use when working on multiple features simultaneously, parallelizing refactors, or orchestrating agent teams. Triggers: parallel agents, multi-agent, worktree, parallel work, agent team, clone, parallel tasks, multiple claudes."
---

> **See also:** `context-engineer` (context window management), `session-handoff` (handoff between sessions).

# Parallel Agents

Run multiple Claude Code instances working on different tasks simultaneously. The key constraint: **each agent gets its own branch or worktree** to prevent write conflicts.

## Models

### 1. Master-Clone (recommended)

Don't build rigid subagent hierarchies. Instead:

1. Put all key context in CLAUDE.md
2. Let the main agent decide when to delegate
3. Agent spawns clones of itself via the `Agent` tool or `claude -p`
4. Each clone inherits full context from CLAUDE.md
5. Merge results back to main

**Why this beats custom subagents**: Custom subagents create context gatekeeping — you're deciding what each agent can see. The master-clone model lets each clone be fully capable and decide what's relevant.

### 2. Branch Isolation

Each parallel agent works on a separate git branch:

```bash
# Agent 1: auth feature
git checkout -b feat/auth
claude -p "Implement JWT authentication in src/auth/"

# Agent 2: API endpoints (separate terminal)
git checkout -b feat/api-endpoints
claude -p "Add CRUD endpoints for users in src/api/"

# Agent 3: tests (separate terminal)
git checkout -b feat/test-coverage
claude -p "Write tests for src/auth/ and src/api/"
```

Merge branches after each agent completes. Resolve conflicts manually.

### 3. Git Worktrees (cleanest isolation)

Worktrees give each agent a full working copy without branch-switching:

```bash
# Create worktrees
git worktree add ../project-auth feat/auth
git worktree add ../project-api feat/api-endpoints
git worktree add ../project-tests feat/test-coverage

# Run agents in each worktree
cd ../project-auth && claude -p "Implement JWT auth..."
cd ../project-api && claude -p "Add CRUD endpoints..."
cd ../project-tests && claude -p "Write test coverage..."

# Clean up when done
git worktree remove ../project-auth
git worktree remove ../project-api
git worktree remove ../project-tests
```

### 4. SDK Parallel Scripting

For batch operations across many files or directories:

```bash
# Parallel rename across directories
for dir in src/modules/*/; do
  claude -p "In $dir, rename all instances of 'oldName' to 'newName'" &
done
wait

# Parallel code review
for file in $(git diff --name-only main); do
  claude -p "Review $file for security issues. Output a summary." &
done
wait
```

## Coordination Patterns

### Shared State via Files

Agents can communicate through files:

```bash
# Agent 1 writes its API contract
claude -p "Define the API contract in docs/api-contract.json"

# Agent 2 reads it and implements
claude -p "Read docs/api-contract.json and implement the client"
```

### Sequential Pipeline

Chain agents where output of one feeds the next:

```bash
# Research → Plan → Implement → Test
claude -p "Research auth patterns, write plan to .claude/auth-plan.md"
claude -p "Read .claude/auth-plan.md, implement it"
claude -p "Read src/auth/, write comprehensive tests"
claude -p "Run all tests, fix any failures"
```

### Fork and Merge

Use Claude Code's `--fork-session` for exploratory branches:

```bash
# Try two approaches in parallel
claude --fork-session  # Approach A
claude --fork-session  # Approach B
# Pick the better result
```

## Safety Rules

- **Never have two agents write to the same file** — use branches or worktrees
- **Always merge back to main** — don't leave orphan branches
- **Each agent gets full CLAUDE.md context** — don't artificially constrain
- **Use `claude -p` for fire-and-forget** — judge the PR, not the process
- **Run tests after merging** — parallel work can create integration issues

## When NOT to Parallelize

- Tasks that depend on each other's output (use sequential pipeline instead)
- Small tasks that take <5 minutes (overhead of coordination exceeds benefit)
- Tasks touching the same files (merge conflicts will cost more than serial execution)
- Debugging (you need to observe sequential cause-and-effect)
