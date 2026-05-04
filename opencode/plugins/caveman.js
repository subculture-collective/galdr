import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { tool } from "@opencode-ai/plugin";

const DEFAULT_MODE = "ultra";
const SESSION_TTL_MS = 30 * 24 * 60 * 60 * 1000;
const STATE_PATH = path.join(
  os.homedir(),
  ".config",
  "opencode",
  "caveman-state.json",
);
const OPENCODE_FLAG_PATH = path.join(
  os.homedir(),
  ".config",
  "opencode",
  ".caveman-active",
);
const LEGACY_FLAG_PATH = path.join(os.homedir(), ".claude", ".caveman-active");

const CANONICAL_MODES = new Set([
  "normal",
  "lite",
  "full",
  "ultra",
  "wenyan-lite",
  "wenyan",
  "wenyan-ultra",
  "commit",
  "review",
]);

const MODE_ALIASES = {
  "": DEFAULT_MODE,
  on: DEFAULT_MODE,
  off: "normal",
  stop: "normal",
  none: "normal",
  clear: "normal",
  normal: "normal",
  default: DEFAULT_MODE,
  wenyanfull: "wenyan",
  "wenyan-full": "wenyan",
  wenyan: "wenyan",
};

function ensureDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

function normalizeMode(rawMode) {
  const value = String(rawMode ?? "")
    .trim()
    .toLowerCase();
  const aliased = MODE_ALIASES[value] ?? value;
  return CANONICAL_MODES.has(aliased) ? aliased : null;
}

function now() {
  return Date.now();
}

function emptyState() {
  return {
    version: 1,
    defaultMode: DEFAULT_MODE,
    sessions: {},
    updatedAt: now(),
  };
}

function cleanState(state) {
  const safe = emptyState();
  const defaultMode = normalizeMode(state?.defaultMode) ?? DEFAULT_MODE;
  safe.defaultMode = defaultMode === "normal" ? DEFAULT_MODE : defaultMode;

  const sessions =
    state?.sessions && typeof state.sessions === "object" ? state.sessions : {};
  const cutoff = now() - SESSION_TTL_MS;

  for (const [sessionID, info] of Object.entries(sessions)) {
    if (!sessionID || typeof sessionID !== "string") continue;
    const mode = normalizeMode(info?.mode);
    const updatedAt = Number(info?.updatedAt || 0);
    if (!mode) continue;
    if (updatedAt && updatedAt < cutoff) continue;
    safe.sessions[sessionID] = {
      mode,
      updatedAt: updatedAt || now(),
    };
  }

  safe.updatedAt = Number(state?.updatedAt || 0) || now();
  return safe;
}

function loadState() {
  try {
    const raw = fs.readFileSync(STATE_PATH, "utf8");
    return cleanState(JSON.parse(raw));
  } catch {
    return emptyState();
  }
}

function saveState(state) {
  const clean = cleanState(state);
  clean.updatedAt = now();
  ensureDir(STATE_PATH);
  fs.writeFileSync(STATE_PATH, `${JSON.stringify(clean, null, 2)}\n`);
}

function writeFlag(mode) {
  const normalized = normalizeMode(mode);
  for (const flagPath of [OPENCODE_FLAG_PATH, LEGACY_FLAG_PATH]) {
    try {
      if (!normalized || normalized === "normal") {
        fs.rmSync(flagPath, { force: true });
        continue;
      }
      ensureDir(flagPath);
      fs.writeFileSync(flagPath, normalized);
    } catch {
      // best-effort only
    }
  }
}

function getSessionMode(state, sessionID) {
  const override = sessionID ? state.sessions?.[sessionID]?.mode : undefined;
  return (
    normalizeMode(override) ?? normalizeMode(state.defaultMode) ?? DEFAULT_MODE
  );
}

function setSessionMode(sessionID, mode) {
  const state = loadState();
  state.sessions[sessionID] = {
    mode,
    updatedAt: now(),
  };
  saveState(state);
  writeFlag(mode);
  return {
    mode,
    defaultMode: state.defaultMode,
    inherited: false,
  };
}

function clearSessionMode(sessionID) {
  const state = loadState();
  delete state.sessions[sessionID];
  saveState(state);
  const effectiveMode = getSessionMode(state, sessionID);
  writeFlag(effectiveMode);
  return {
    mode: effectiveMode,
    defaultMode: state.defaultMode,
    inherited: true,
  };
}

function setDefaultMode(mode) {
  const state = loadState();
  state.defaultMode = mode === "normal" ? DEFAULT_MODE : mode;
  saveState(state);
  writeFlag(state.defaultMode);
  return state.defaultMode;
}

function removeSession(sessionID) {
  const state = loadState();
  if (!state.sessions[sessionID]) return;
  delete state.sessions[sessionID];
  saveState(state);
}

function currentStatus(sessionID) {
  const state = loadState();
  const inherited = !state.sessions?.[sessionID];
  return {
    mode: getSessionMode(state, sessionID),
    defaultMode: state.defaultMode,
    inherited,
  };
}

function listModes() {
  return [
    "normal",
    "lite",
    "full",
    "ultra",
    "wenyan-lite",
    "wenyan",
    "wenyan-ultra",
    "commit",
    "review",
  ];
}

function extractText(parts) {
  return parts
    .filter((part) => part.type === "text" && typeof part.text === "string")
    .map((part) => part.text)
    .join("\n")
    .trim();
}

function buildBasePrompt(level, extraRules = []) {
  const lines = [
    `CAVEMAN MODE ACTIVE: ${level}.`,
    "Speak terse like smart caveman. Keep full technical substance. Kill fluff.",
    "Drop filler, hedging, pleasantries, and extra throat-clearing.",
    "Keep code blocks, commands, file paths, env vars, errors, and exact quoted text unchanged.",
    "Commit messages: use Conventional Commits, why over what, subject <=50 chars when possible.",
    "Code review findings: findings first. One line each: <file>:L<line>: <problem>. <fix>.",
    "For destructive actions, security warnings, or confusing multi-step guidance, switch to plain clear prose.",
    ...extraRules,
  ];
  return lines.join(" ");
}

function buildModePrompt(mode) {
  switch (mode) {
    case "normal":
      return "";
    case "lite":
      return buildBasePrompt("LITE", [
        "Keep full sentences and grammar. Be professional, tight, and low-fluff.",
      ]);
    case "full":
      return buildBasePrompt("FULL", [
        "Fragments OK. Drop articles when safe. Prefer short everyday words.",
      ]);
    case "ultra":
      return buildBasePrompt("ULTRA", [
        "Maximum compression. Abbrev OK: DB, auth, config, req, res, fn, impl.",
        "Arrows OK for causality: X -> Y.",
        "One word when one word enough.",
      ]);
    case "wenyan-lite":
      return buildBasePrompt("WENYAN-LITE", [
        "Use semi-classical Chinese flavor with readable grammar. Keep technical tokens exact.",
      ]);
    case "wenyan":
      return buildBasePrompt("WENYAN", [
        "Use classical Chinese compression where practical. Keep technical tokens exact.",
      ]);
    case "wenyan-ultra":
      return buildBasePrompt("WENYAN-ULTRA", [
        "Use maximum classical compression. Keep technical tokens exact. Be extremely terse.",
      ]);
    case "commit":
      return [
        "CAVEMAN COMMIT MODE ACTIVE.",
        "Write commit messages terse and exact.",
        "Format: <type>(<scope>): <imperative summary>. Scope optional.",
        "Use Conventional Commit types: feat, fix, refactor, perf, docs, test, chore, build, ci, style, revert.",
        "Subject <=50 chars when possible, hard cap 72, no trailing period.",
        "Why over what. Add body only when why is not obvious, or for breaking/security/migration/revert context.",
        "No AI attribution, no fluff, no restating filenames.",
      ].join(" ");
    case "review":
      return [
        "CAVEMAN REVIEW MODE ACTIVE.",
        "Do code review in terse finding-first style.",
        "One line per finding: <file>:L<line>: <problem>. <fix>.",
        "Prioritize bugs, regressions, risk, and missing tests.",
        "If no findings, say so explicitly and mention residual risk or test gaps.",
        "Use plain paragraphs only for security issues or large architectural disagreement.",
      ].join(" ");
    default:
      return buildModePrompt(DEFAULT_MODE);
  }
}

const STOP_REGEX =
  /\b(stop caveman|normal mode|disable caveman|turn off caveman)\b/i;
const RESUME_REGEX = /\b(resume caveman|enable caveman|caveman on)\b/i;

export const id = "caveman";

export const server = async () => {
  return {
    tool: {
      caveman_mode: tool({
        description:
          "Manage caveman mode for the current OpenCode session or global default.",
        args: {
          action: tool.schema.enum([
            "get",
            "set",
            "clear",
            "set-default",
            "reset-default",
            "list",
          ]),
          mode: tool.schema
            .enum([
              "normal",
              "lite",
              "full",
              "ultra",
              "wenyan-lite",
              "wenyan",
              "wenyan-ultra",
              "commit",
              "review",
            ])
            .optional(),
        },
        async execute(args, context) {
          const sessionID = context.sessionID;
          switch (args.action) {
            case "list": {
              context.metadata({ title: "Caveman modes" });
              return `Modes: ${listModes().join(", ")}`;
            }
            case "reset-default": {
              const defaultMode = setDefaultMode(DEFAULT_MODE);
              const status = currentStatus(sessionID);
              context.metadata({
                title: "Reset caveman default",
                metadata: status,
              });
              return `Default ${defaultMode}. Session ${status.mode}.`;
            }
            case "set-default": {
              if (!args.mode) throw new Error("mode required for set-default");
              const defaultMode = setDefaultMode(args.mode);
              const status = currentStatus(sessionID);
              context.metadata({
                title: "Set caveman default",
                metadata: status,
              });
              return `Default ${defaultMode}. Session ${status.mode}.`;
            }
            case "clear": {
              const status = clearSessionMode(sessionID);
              context.metadata({
                title: "Clear caveman session override",
                metadata: status,
              });
              return `Session inherits default. Active ${status.mode}. Default ${status.defaultMode}.`;
            }
            case "set": {
              if (!args.mode) throw new Error("mode required for set");
              const status = setSessionMode(sessionID, args.mode);
              context.metadata({
                title: "Set caveman session mode",
                metadata: status,
              });
              return `Session ${status.mode}. Default ${status.defaultMode}.`;
            }
            case "get":
            default: {
              const status = currentStatus(sessionID);
              context.metadata({ title: "Caveman status", metadata: status });
              return `Session ${status.mode}${status.inherited ? " (inherits default)" : ""}. Default ${status.defaultMode}.`;
            }
          }
        },
      }),
    },

    event: async ({ event }) => {
      if (event.type === "session.created") {
        const state = loadState();
        writeFlag(state.defaultMode);
        return;
      }

      if (event.type === "session.deleted") {
        const sessionID =
          event.properties?.sessionID || event.properties?.info?.id;
        if (sessionID) removeSession(sessionID);
      }
    },

    "chat.message": async (input, output) => {
      const prompt = extractText(output.parts);
      if (!prompt) return;

      if (STOP_REGEX.test(prompt)) {
        setSessionMode(input.sessionID, "normal");
        return;
      }

      if (RESUME_REGEX.test(prompt)) {
        setSessionMode(input.sessionID, loadState().defaultMode);
      }
    },

    "experimental.chat.system.transform": async (input, output) => {
      const state = loadState();
      const mode = getSessionMode(state, input.sessionID);
      writeFlag(mode);
      const prompt = buildModePrompt(mode);
      if (!prompt) return;
      output.system.push(prompt);
    },
  };
};

export default { id, server };
