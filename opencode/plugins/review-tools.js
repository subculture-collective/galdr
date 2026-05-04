import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { tool } from "@opencode-ai/plugin";

import { listFilesRecursive as walkFiles } from "./_shared.mjs";

const AUTO_REVIEW_STATE = path.join(
  os.homedir(),
  ".config",
  "opencode",
  "review-tools-state.json",
);
const CODE_EXTS = new Set([
  ".ts",
  ".tsx",
  ".js",
  ".jsx",
  ".mjs",
  ".cjs",
  ".py",
  ".go",
  ".rs",
]);
const TEST_HINTS = ["test", "tests", "spec", "__tests__", "__specs__"];

function ensureDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

function loadReviewState() {
  try {
    const parsed = JSON.parse(fs.readFileSync(AUTO_REVIEW_STATE, "utf8"));
    return {
      autoReview: Boolean(parsed.autoReview),
      lastNotifiedAt:
        typeof parsed.lastNotifiedAt === "string" ?
          parsed.lastNotifiedAt
        : null,
      lastNotifiedFingerprint:
        typeof parsed.lastNotifiedFingerprint === "string" ?
          parsed.lastNotifiedFingerprint
        : "",
    };
  } catch {
    return {
      autoReview: false,
      lastNotifiedAt: null,
      lastNotifiedFingerprint: "",
    };
  }
}

function saveReviewState(state) {
  ensureDir(AUTO_REVIEW_STATE);
  fs.writeFileSync(AUTO_REVIEW_STATE, `${JSON.stringify(state, null, 2)}\n`);
}

function shellEscape(value) {
  return `'${String(value).replace(/'/g, `'\\''`)}'`;
}

async function runShell($, command) {
  const result = await $`bash -lc ${command}`.quiet();
  const stdout =
    typeof result.stdout === "string" ?
      result.stdout.trim()
    : String(result.stdout || "").trim();
  const stderr =
    typeof result.stderr === "string" ?
      result.stderr.trim()
    : String(result.stderr || "").trim();
  return stdout || stderr || "";
}

function listFilesRecursive(root) {
  return walkFiles(root);
}

function fingerprint(value) {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i += 1) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(16);
}

async function getChangedFiles($, directory, base) {
  const command =
    base ?
      `git diff --name-only ${shellEscape(base)}...HEAD`
    : `git diff --name-only && printf '\n' && git diff --cached --name-only`;
  const output = await runShell($, command);
  const seen = new Set();
  return output
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .filter((line) => {
      if (seen.has(line)) return false;
      seen.add(line);
      return fs.existsSync(path.join(directory, line));
    });
}

function summarizeComplexitySignals(directory, filesOrPath) {
  const files =
    Array.isArray(filesOrPath) ? filesOrPath : (
      listFilesRecursive(
        filesOrPath ? path.join(directory, filesOrPath) : directory,
      )
    );
  const results = [];
  for (const full of files) {
    if (!CODE_EXTS.has(path.extname(full))) continue;
    let text = "";
    try {
      text = fs.readFileSync(full, "utf8");
    } catch {
      continue;
    }
    const rel = path.relative(directory, full);
    const lines = text.split(/\r?\n/).length;
    const branches = (
      text.match(/\b(if|else if|switch|case|catch|for|while)\b|\?/g) || []
    ).length;
    const funcs = (text.match(/\bfunction\b|=>|\bdef\b|\bfunc\b/g) || [])
      .length;
    const score = lines + branches * 12 + funcs * 4;
    if (lines >= 250 || branches >= 12 || score >= 450) {
      results.push(
        `${rel}: lines=${lines}, branches=${branches}, funcs=${funcs}, score=${score}`,
      );
    }
  }
  return results.sort((a, b) => {
    const scoreA = Number(a.match(/score=(\d+)/)?.[1] || 0);
    const scoreB = Number(b.match(/score=(\d+)/)?.[1] || 0);
    return scoreB - scoreA || a.localeCompare(b);
  });
}

function findCoverageGaps(directory, relFiles) {
  const files =
    relFiles?.length ?
      relFiles
        .map((rel) => path.join(directory, rel))
        .filter((full) => fs.existsSync(full))
    : listFilesRecursive(directory);
  const testKeys = new Set();
  for (const full of files) {
    const rel = path.relative(directory, full);
    const lower = rel.toLowerCase();
    const ext = path.extname(full);
    if (!CODE_EXTS.has(ext)) continue;
    if (!TEST_HINTS.some((hint) => lower.includes(hint))) continue;
    const base = path.basename(rel, ext).replace(/\.(test|spec)$/i, "");
    testKeys.add(base);
  }

  const gaps = [];
  for (const full of files) {
    const rel = path.relative(directory, full);
    const lower = rel.toLowerCase();
    const ext = path.extname(full);
    if (!CODE_EXTS.has(ext)) continue;
    if (TEST_HINTS.some((hint) => lower.includes(hint))) continue;
    if (lower.includes("/scripts/") || lower.includes("/bin/")) continue;
    const base = path.basename(rel, ext);
    if (!testKeys.has(base)) gaps.push(rel);
  }
  return gaps.sort();
}

function findDeadExports(directory, filesOrPath) {
  const files = (
    Array.isArray(filesOrPath) ? filesOrPath : (
      listFilesRecursive(
        filesOrPath ? path.join(directory, filesOrPath) : directory,
      )
    )).filter((full) =>
    [".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"].includes(path.extname(full)),
  );
  const content = new Map();
  for (const full of files) {
    try {
      content.set(full, fs.readFileSync(full, "utf8"));
    } catch {
      content.set(full, "");
    }
  }

  const hits = [];
  const exportRe =
    /export\s+(?:async\s+)?(?:function|const|class|type|interface|enum)\s+([A-Za-z0-9_]+)/g;
  for (const [full, text] of content.entries()) {
    const rel = path.relative(directory, full);
    let match;
    while ((match = exportRe.exec(text))) {
      const name = match[1];
      if (!name) continue;
      const localUse =
        text.replace(match[0], "").match(new RegExp(`\\b${name}\\b`, "g"))
          ?.length || 0;
      let importUse = 0;
      for (const [other, otherText] of content.entries()) {
        if (other === full) continue;
        if (new RegExp(`\\b${name}\\b`).test(otherText)) importUse += 1;
      }
      if (importUse === 0 && localUse <= 1) hits.push(`${rel}: ${name}`);
    }
  }
  return hits.sort();
}

function summarizeLargeFiles(directory, maybePath) {
  const root = maybePath ? path.join(directory, maybePath) : directory;
  const out = [];
  function walk(current) {
    let entries = [];
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      if (
        ["node_modules", ".git", "dist", "build", ".next"].includes(entry.name)
      )
        continue;
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      let lineCount = 0;
      try {
        lineCount = fs.readFileSync(full, "utf8").split(/\r?\n/).length;
      } catch {
        continue;
      }
      if (lineCount > 500)
        out.push(`${path.relative(directory, full)}: ${lineCount} lines`);
    }
  }
  walk(root);
  return out.sort();
}

function classifyChangedFiles(relFiles) {
  const buckets = {
    code: [],
    tests: [],
    docs: [],
    config: [],
  };
  for (const rel of relFiles) {
    const lower = rel.toLowerCase();
    if (TEST_HINTS.some((hint) => lower.includes(hint)))
      buckets.tests.push(rel);
    else if (lower.endsWith(".md") || lower.startsWith("docs/"))
      buckets.docs.push(rel);
    else if (
      lower.endsWith(".json") ||
      lower.endsWith(".yaml") ||
      lower.endsWith(".yml") ||
      lower.endsWith(".toml")
    )
      buckets.config.push(rel);
    else buckets.code.push(rel);
  }
  return buckets;
}

function recommendReviewTools(buckets) {
  const tools = ["review_diff"];
  if (buckets.code.length) tools.push("review_summary", "review_complexity");
  if (buckets.tests.length === 0 && buckets.code.length)
    tools.push("review_coverage_gaps");
  if (buckets.docs.length) tools.push("review_todos");
  return [...new Set(tools)];
}

export const id = "review-tools";

export const server = async ({ $, directory, client }) => {
  return {
    tool: {
      review_diff: tool({
        description: "Show staged and unstaged diff with summary stats.",
        args: {
          base: tool.schema.string().optional().describe("Optional base ref"),
        },
        async execute(args, context) {
          const output =
            args.base ?
              await runShell(
                $,
                `git diff --stat ${shellEscape(args.base)}...HEAD && printf '\n' && git diff ${shellEscape(args.base)}...HEAD`,
              )
            : await runShell(
                $,
                `git diff --stat && printf '\n' && git diff && printf '\n\n-- staged --\n' && git diff --cached --stat && printf '\n' && git diff --cached`,
              );
          context.metadata({
            title: "Review diff",
            metadata: { base: args.base || null },
          });
          return output || "No diff.";
        },
      }),
      review_todos: tool({
        description: "Scan repo for TODO, FIXME, HACK, and XXX markers.",
        args: {},
        async execute(_args, context) {
          const output = await runShell(
            $,
            `rg -n --hidden --glob '!.git' --glob '!node_modules' --glob '!dist' --glob '!build' 'TODO|FIXME|HACK|XXX'`,
          );
          context.metadata({ title: "Review todos" });
          return output || "No TODO-like markers found.";
        },
      }),
      review_complexity: tool({
        description: "Surface long files and coarse complexity hotspots.",
        args: {
          path: tool.schema.string().optional().describe("Optional subtree"),
          scope: tool.schema
            .enum(["changed", "all"])
            .optional()
            .describe("Analyze changed files or whole repo"),
          base: tool.schema
            .string()
            .optional()
            .describe("Optional base ref for changed scope"),
        },
        async execute(args, context) {
          const changed =
            args.scope === "changed" ?
              await getChangedFiles($, directory, args.base)
            : null;
          const selectedFiles =
            changed?.length ?
              changed.map((rel) => path.join(directory, rel))
            : undefined;
          const largeFiles = summarizeLargeFiles(directory, args.path);
          const scored = summarizeComplexitySignals(
            directory,
            selectedFiles || args.path,
          ).slice(0, 25);
          const nesting = await runShell(
            $,
            `rg -n --hidden --glob '!.git' --glob '!node_modules' --glob '!dist' --glob '!build' 'if .*if|for .*for|while .*while|\? .*:' ${args.path ? shellEscape(args.path) : "."}`,
          );
          context.metadata({
            title: "Review complexity",
            metadata: {
              path: args.path || ".",
              scope: args.scope || "all",
              base: args.base || null,
            },
          });
          return [
            "Large files:",
            largeFiles.length ? largeFiles.join("\n") : "none",
            "",
            "Scored hotspots:",
            scored.length ? scored.join("\n") : "none",
            "",
            "Potential nesting hotspots:",
            nesting || "none",
          ].join("\n");
        },
      }),
      review_dead_exports: tool({
        description: "Find exported symbols with no obvious imports.",
        args: {
          path: tool.schema.string().optional().describe("Optional subtree"),
          scope: tool.schema
            .enum(["changed", "all"])
            .optional()
            .describe("Analyze changed files or whole repo"),
          base: tool.schema
            .string()
            .optional()
            .describe("Optional base ref for changed scope"),
        },
        async execute(args, context) {
          const changed =
            args.scope === "changed" ?
              await getChangedFiles($, directory, args.base)
            : null;
          const selectedFiles =
            changed?.length ?
              changed.map((rel) => path.join(directory, rel))
            : undefined;
          const output = findDeadExports(
            directory,
            selectedFiles || args.path,
          ).join("\n");
          context.metadata({
            title: "Review dead exports",
            metadata: {
              path: args.path || ".",
              scope: args.scope || "all",
              base: args.base || null,
            },
          });
          return output || "No obvious dead exports found.";
        },
      }),
      review_search: tool({
        description: "Search with small context windows for review work.",
        args: {
          pattern: tool.schema.string().describe("Pattern to search"),
        },
        async execute(args, context) {
          const output = await runShell(
            $,
            `rg -n -C 3 --hidden --glob '!.git' --glob '!node_modules' --glob '!dist' --glob '!build' ${shellEscape(args.pattern)}`,
          );
          context.metadata({
            title: "Review search",
            metadata: { pattern: args.pattern },
          });
          return output || "No matches found.";
        },
      }),
      review_coverage_gaps: tool({
        description: "Find source files with no nearby test file.",
        args: {
          scope: tool.schema
            .enum(["changed", "all"])
            .optional()
            .describe("Analyze changed files or whole repo"),
          base: tool.schema
            .string()
            .optional()
            .describe("Optional base ref for changed scope"),
        },
        async execute(args, context) {
          const changed =
            args.scope === "changed" ?
              await getChangedFiles($, directory, args.base)
            : null;
          const output = findCoverageGaps(directory, changed || undefined).join(
            "\n",
          );
          context.metadata({
            title: "Review coverage gaps",
            metadata: { scope: args.scope || "all", base: args.base || null },
          });
          return output || "No obvious coverage gaps found.";
        },
      }),
      review_changed_files: tool({
        description:
          "Summarize changed files, risk buckets, and suggested review tools.",
        args: {
          base: tool.schema.string().optional().describe("Optional base ref"),
        },
        async execute(args, context) {
          const changed = await getChangedFiles($, directory, args.base);
          const buckets = classifyChangedFiles(changed);
          const tools = recommendReviewTools(buckets);
          context.metadata({
            title: "Review changed files",
            metadata: {
              base: args.base || null,
              count: changed.length,
              buckets,
              tools,
            },
          });
          return [
            `Changed files: ${changed.length}`,
            changed.length ? changed.join("\n") : "none",
            "",
            `Code: ${buckets.code.length}`,
            `Tests: ${buckets.tests.length}`,
            `Docs: ${buckets.docs.length}`,
            `Config: ${buckets.config.length}`,
            "",
            `Suggested tools: ${tools.join(", ")}`,
          ].join("\n");
        },
      }),
      review_auto: tool({
        description: "Enable, disable, or inspect automatic review reminders.",
        args: {
          mode: tool.schema
            .enum(["on", "off", "status"])
            .optional()
            .describe("Set auto-review on/off or inspect status"),
        },
        async execute(args, context) {
          const state = loadReviewState();
          const mode = args.mode || "status";
          if (mode === "on") state.autoReview = true;
          if (mode === "off") state.autoReview = false;
          saveReviewState(state);
          context.metadata({ title: "Review auto mode", metadata: state });
          return `Auto-review ${state.autoReview ? "on" : "off"}.`;
        },
      }),
      review_summary: tool({
        description:
          "Single-shot review summary with diff, todos, and hotspots.",
        args: {
          base: tool.schema.string().optional().describe("Optional base ref"),
          scope: tool.schema
            .enum(["changed", "all"])
            .optional()
            .describe("Analyze changed files or whole repo"),
        },
        async execute(args, context) {
          const diff =
            args.base ?
              await runShell(
                $,
                `git diff --stat ${shellEscape(args.base)}...HEAD && printf '\n' && git diff ${shellEscape(args.base)}...HEAD`,
              )
            : await runShell(
                $,
                `git diff --stat && printf '\n' && git diff --cached --stat`,
              );
          const changed =
            args.scope === "changed" || args.base ?
              await getChangedFiles($, directory, args.base)
            : null;
          const todos = await runShell(
            $,
            `rg -n --hidden --glob '!.git' --glob '!node_modules' --glob '!dist' --glob '!build' 'TODO|FIXME|HACK|XXX'`,
          );
          const selectedFiles =
            changed?.length ?
              changed.map((rel) => path.join(directory, rel))
            : undefined;
          const hotspots = summarizeComplexitySignals(
            directory,
            selectedFiles || undefined,
          )
            .slice(0, 15)
            .join("\n");
          const gaps = findCoverageGaps(directory, changed || undefined)
            .slice(0, 20)
            .join("\n");
          const exports = findDeadExports(directory, selectedFiles || undefined)
            .slice(0, 20)
            .join("\n");
          context.metadata({
            title: "Review summary",
            metadata: {
              base: args.base || null,
              scope: args.scope || (args.base ? "changed" : "all"),
            },
          });
          return [
            "Diff:",
            diff || "none",
            "",
            "TODOs:",
            todos || "none",
            "",
            "Hotspots:",
            hotspots || "none",
            "",
            "Coverage gaps:",
            gaps || "none",
            "",
            "Possible dead exports:",
            exports || "none",
          ].join("\n");
        },
      }),
      review_pr_ready: tool({
        description:
          "High-signal merge-readiness summary for current worktree or branch diff.",
        args: {
          base: tool.schema.string().optional().describe("Optional base ref"),
        },
        async execute(args, context) {
          const changed = await getChangedFiles($, directory, args.base);
          const buckets = classifyChangedFiles(changed);
          const diff =
            args.base ?
              await runShell(
                $,
                `git diff --stat ${shellEscape(args.base)}...HEAD`,
              )
            : await runShell(
                $,
                `git diff --stat && printf '\n' && git diff --cached --stat`,
              );
          const hotspots = summarizeComplexitySignals(
            directory,
            changed.map((rel) => path.join(directory, rel)),
          ).slice(0, 10);
          const gaps = findCoverageGaps(directory, changed).slice(0, 10);
          const exports = findDeadExports(
            directory,
            changed.map((rel) => path.join(directory, rel)),
          ).slice(0, 10);
          const risk =
            hotspots.length || gaps.length || exports.length ?
              "with fixes"
            : "ready";
          context.metadata({
            title: "Review PR ready",
            metadata: {
              base: args.base || null,
              changed: changed.length,
              risk,
              buckets,
            },
          });
          return [
            `Verdict: ${risk}`,
            `Changed files: ${changed.length}`,
            diff || "No diff.",
            "",
            `Risk buckets: code=${buckets.code.length}, tests=${buckets.tests.length}, docs=${buckets.docs.length}, config=${buckets.config.length}`,
            `Hotspots: ${hotspots.length ? hotspots.join(" | ") : "none"}`,
            `Coverage gaps: ${gaps.length ? gaps.join(" | ") : "none"}`,
            `Possible dead exports: ${exports.length ? exports.join(" | ") : "none"}`,
          ].join("\n");
        },
      }),
    },

    event: async ({ event }) => {
      if (event.type !== "session.idle") return;
      const reviewState = loadReviewState();
      if (!reviewState.autoReview) return;
      const diff = await runShell(
        $,
        `git diff --name-only && printf '\n' && git diff --cached --name-only`,
      );
      if (!diff.trim()) return;
      const currentFingerprint = fingerprint(diff.trim());
      if (reviewState.lastNotifiedFingerprint === currentFingerprint) return;
      try {
        await client.session.prompt({
          path: { id: event.properties.sessionID },
          body: {
            noReply: true,
            parts: [
              {
                type: "text",
                text: `<review-summary>Changed files detected. Run review_changed_files, review_summary, or review_pr_ready before finalizing. Files:\n${diff.trim()}\n</review-summary>`,
              },
            ],
          },
        });
        reviewState.lastNotifiedAt = new Date().toISOString();
        reviewState.lastNotifiedFingerprint = currentFingerprint;
        saveReviewState(reviewState);
      } catch {
        // best effort only
      }
    },
  };
};

export default { id, server };
