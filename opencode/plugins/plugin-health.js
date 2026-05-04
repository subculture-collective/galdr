import { Database } from "bun:sqlite";
import path from "node:path";

import { tool } from "@opencode-ai/plugin";

import {
  OPENCODE_CONFIG_DIR,
  listCommandFiles,
  listLocalPluginFiles,
  findBinaryOnPath,
  loadJson,
  observeProjectDbPath,
  resolveCachedPackageEntry,
} from "./_shared.mjs";

const FILES = {
  caveman: path.join(OPENCODE_CONFIG_DIR, "caveman-state.json"),
  review: path.join(OPENCODE_CONFIG_DIR, "review-tools-state.json"),
  scoutqa: path.join(OPENCODE_CONFIG_DIR, "scoutqa-state.json"),
  observe: path.join(OPENCODE_CONFIG_DIR, "observe-state.json"),
  evolution: path.join(OPENCODE_CONFIG_DIR, "evolution", "sessions.json"),
};

function readPluginStates() {
  const caveman = loadJson(FILES.caveman, {
    defaultMode: "ultra",
    sessions: {},
  });
  const review = loadJson(FILES.review, { autoReview: false });
  const scoutqa = loadJson(FILES.scoutqa, {
    autoRetest: false,
    targetUrl: "",
    pendingRetest: false,
    testType: "smoke",
    lastRun: null,
  });
  const evolution = loadJson(FILES.evolution, {});
  return { caveman, review, scoutqa, evolution };
}

function readObserveState(directory) {
  const dbPath = observeProjectDbPath(directory);
  try {
    const db = new Database(dbPath, { create: false, readonly: true });
    const counts =
      db
        .query(
          `
      select
        sum(case when active = 1 then 1 else 0 end) as active_count,
        sum(case when active = 1 and pending = 1 then 1 else 0 end) as pending_count,
        min(case when active = 1 then next_check_at end) as next_check_at
      from loops
    `,
        )
        .get() || {};
    db.close();
    return {
      dbPath,
      activeCount: Number(counts.active_count || 0),
      pendingCount: Number(counts.pending_count || 0),
      nextCheckAt: counts.next_check_at || null,
    };
  } catch {
    return {
      dbPath,
      activeCount: 0,
      pendingCount: 0,
      nextCheckAt: null,
    };
  }
}

function buildStatus(directory) {
  const plugins = listLocalPluginFiles();
  const commands = listCommandFiles();
  const states = readPluginStates();
  const observe = readObserveState(directory);
  const scoutqaBin = findBinaryOnPath("scoutqa");
  const ghBin = findBinaryOnPath("gh");
  const skillsModule = resolveCachedPackageEntry(
    "opencode-agent-skills",
    path.join("src", "skills.ts"),
  );

  return {
    plugins,
    commands,
    binaries: {
      scoutqa: scoutqaBin,
      gh: ghBin,
    },
    packageResolution: {
      opencodeAgentSkills: skillsModule,
    },
    observe,
    states,
  };
}

function renderStatus(status) {
  return [
    `Plugins: ${status.plugins.length} (${status.plugins.join(", ")})`,
    `Commands: ${status.commands.length}`,
    `Caveman default: ${status.states.caveman.defaultMode || "unknown"}`,
    `Auto-review: ${status.states.review.autoReview ? "on" : "off"}`,
    `ScoutQA auto-retest: ${status.states.scoutqa.autoRetest ? "on" : "off"}; binary=${status.binaries.scoutqa || "missing"}`,
    `Observe loops: active=${status.observe.activeCount} pending=${status.observe.pendingCount} next=${status.observe.nextCheckAt || "none"}`,
    `Evolution active sessions: ${Object.keys(status.states.evolution || {}).length}`,
    `gh binary: ${status.binaries.gh || "missing"}`,
    `opencode-agent-skills source: ${status.packageResolution.opencodeAgentSkills || "missing"}`,
  ].join("\n");
}

function renderDoctor(status) {
  const issues = [];
  if (!status.binaries.gh)
    issues.push("gh missing. GitHub plugin tools will fail.");
  if (!status.binaries.scoutqa)
    issues.push(
      "scoutqa missing. ScoutQA run/status tools limited to stored local artifacts.",
    );
  if (!status.packageResolution.opencodeAgentSkills)
    issues.push(
      "opencode-agent-skills source unresolved. skill-injector fallback discovery will be used.",
    );
  if (status.plugins.length === 0)
    issues.push("No local plugins found in ~/.config/opencode/plugins.");
  if (!status.commands.includes("plugin-status.md"))
    issues.push("plugin-status command wrapper missing.");
  if (!status.commands.includes("plugin-doctor.md"))
    issues.push("plugin-doctor command wrapper missing.");
  return issues.length ?
      issues.map((issue) => `- ${issue}`).join("\n")
    : "No obvious plugin health issues.";
}

export const id = "plugin-health";

export const server = async ({ directory }) => {
  return {
    tool: {
      plugin_status: tool({
        description: "Show local plugin fleet status and key runtime state.",
        args: {},
        async execute(_args, context) {
          const status = buildStatus(directory);
          context.metadata({ title: "Plugin status", metadata: status });
          return renderStatus(status);
        },
      }),
      plugin_doctor: tool({
        description:
          "Check local plugin health, dependencies, and obvious misconfigurations.",
        args: {},
        async execute(_args, context) {
          const status = buildStatus(directory);
          const report = renderDoctor(status);
          context.metadata({ title: "Plugin doctor", metadata: status });
          return [renderStatus(status), "", "Issues:", report].join("\n");
        },
      }),
    },
  };
};

export default { id, server };
