import { Database } from "bun:sqlite";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { tool } from "@opencode-ai/plugin";

import { ensureDir, observeProjectDbPath } from "./_shared.mjs";

const DEFAULT_INTERVAL = 30;
const DEFAULT_MAX_CYCLES = 10;
const DEFAULT_HISTORY_LIMIT = 10;
const LOCK_TTL_MS = 5 * 60 * 1000;
const LOOP_RETENTION_DAYS = 30;
const EVENT_RETENTION_DAYS = 45;
const LEGACY_STATE_PATH = path.join(
  os.homedir(),
  ".config",
  "opencode",
  "observe-state.json",
);
const LEGACY_HISTORY_PATH = path.join(
  os.homedir(),
  ".config",
  "opencode",
  "observe-history.jsonl",
);

function nowIso() {
  return new Date().toISOString();
}

function futureIso(seconds) {
  return new Date(Date.now() + seconds * 1000).toISOString();
}

function makeLoopID() {
  return `obs-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function openObserveDb(directory) {
  const dbPath = observeProjectDbPath(directory);
  ensureDir(dbPath);
  const db = new Database(dbPath);
  db.exec("PRAGMA journal_mode = WAL;");
  db.exec("PRAGMA busy_timeout = 5000;");
  db.exec(`
    create table if not exists loops (
      id text primary key,
      session_id text,
      target text not null,
      interval_secs integer not null,
      max_cycles integer not null,
      success_criteria text not null default '',
      active integer not null default 1,
      cycle_count integer not null default 0,
      pending integer not null default 0,
      last_result text not null default 'scheduled',
      last_summary text not null default '',
      last_trigger text not null default '',
      started_at text not null,
      last_checked_at text,
      next_check_at text,
      completed_at text,
      lock_expires_at integer,
      created_at text not null,
      updated_at text not null
    );
    create index if not exists idx_loops_active_session on loops(active, session_id, next_check_at);
    create table if not exists loop_events (
      id integer primary key autoincrement,
      loop_id text not null,
      at text not null,
      type text not null,
      session_id text,
      result text,
      summary text,
      details text
    );
    create index if not exists idx_loop_events_loop_id on loop_events(loop_id, at desc);
    create table if not exists settings (
      key text primary key,
      value text not null
    );
  `);
  migrateLegacyObserveState(db);
  pruneObserveDb(db);
  return { db, dbPath };
}

function settingValue(db, key) {
  return (
    db.query(`select value from settings where key = ?`).get(key)?.value || null
  );
}

function setSetting(db, key, value) {
  db.query(
    `insert into settings(key, value) values (?, ?) on conflict(key) do update set value = excluded.value`,
  ).run(key, value);
}

function migrateLegacyObserveState(db) {
  if (settingValue(db, "legacy_migrated") === "1") return;
  if (!fs.existsSync(LEGACY_STATE_PATH)) {
    setSetting(db, "legacy_migrated", "1");
    return;
  }
  try {
    const legacy = JSON.parse(fs.readFileSync(LEGACY_STATE_PATH, "utf8"));
    if (legacy?.active && legacy.target) {
      const loopID = makeLoopID();
      const startedAt = legacy.startedAt || nowIso();
      const nextCheckAt =
        legacy.nextCheckAt ||
        futureIso(Number(legacy.interval || DEFAULT_INTERVAL));
      db.query(
        `
        insert or ignore into loops (
          id, session_id, target, interval_secs, max_cycles, success_criteria,
          active, cycle_count, pending, last_result, last_summary, last_trigger,
          started_at, last_checked_at, next_check_at, completed_at, created_at, updated_at
        ) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      `,
      ).run(
        loopID,
        legacy.sessionID || null,
        legacy.target,
        Number(legacy.interval || DEFAULT_INTERVAL),
        Number(legacy.maxCycles || DEFAULT_MAX_CYCLES),
        legacy.criteria || "",
        legacy.active ? 1 : 0,
        Number(legacy.cycle || 0),
        legacy.pending ? 1 : 0,
        legacy.lastResult || "scheduled",
        legacy.lastSummary || "",
        legacy.lastTrigger || "legacy-migrate",
        startedAt,
        legacy.lastCheckedAt || null,
        nextCheckAt,
        legacy.active ? null : legacy.lastCheckedAt || nowIso(),
        startedAt,
        nowIso(),
      );
      insertEvent(db, {
        loopID,
        type: "legacy-migrate",
        sessionID: legacy.sessionID || null,
        result: legacy.lastResult || null,
        summary:
          legacy.lastSummary || "Imported from legacy observe-state.json",
        at: nowIso(),
      });
    }

    if (fs.existsSync(LEGACY_HISTORY_PATH)) {
      const lines = fs
        .readFileSync(LEGACY_HISTORY_PATH, "utf8")
        .split(/\r?\n/)
        .filter(Boolean)
        .slice(-50);
      for (const line of lines) {
        try {
          const entry = JSON.parse(line);
          insertEvent(db, {
            loopID: "legacy-history",
            type: entry.type || "legacy-history",
            sessionID: entry.sessionID || null,
            result: entry.result || null,
            summary: entry.summary || entry.target || "Imported legacy history",
            at: entry.at || nowIso(),
            details: entry,
          });
        } catch {
          // ignore bad lines
        }
      }
    }
  } catch {
    // ignore migration failures; do not block plugin
  }
  setSetting(db, "legacy_migrated", "1");
}

function pruneObserveDb(db) {
  const loopCutoff = new Date(
    Date.now() - LOOP_RETENTION_DAYS * 24 * 60 * 60 * 1000,
  ).toISOString();
  const eventCutoff = new Date(
    Date.now() - EVENT_RETENTION_DAYS * 24 * 60 * 60 * 1000,
  ).toISOString();
  db.query(`delete from loop_events where at < ?`).run(eventCutoff);
  db.query(
    `delete from loops where active = 0 and coalesce(completed_at, updated_at, created_at) < ?`,
  ).run(loopCutoff);
  db.query(
    `update loops set pending = 0, lock_expires_at = null where pending = 1 and lock_expires_at is not null and lock_expires_at < ?`,
  ).run(Date.now());
}

function pruneObserveDbWithCounts(db) {
  const before = {
    loops: Number(
      db.query(`select count(*) as count from loops`).get()?.count || 0,
    ),
    events: Number(
      db.query(`select count(*) as count from loop_events`).get()?.count || 0,
    ),
  };
  pruneObserveDb(db);
  const after = {
    loops: Number(
      db.query(`select count(*) as count from loops`).get()?.count || 0,
    ),
    events: Number(
      db.query(`select count(*) as count from loop_events`).get()?.count || 0,
    ),
  };
  return {
    loopsRemoved: before.loops - after.loops,
    eventsRemoved: before.events - after.events,
    remainingLoops: after.loops,
    remainingEvents: after.events,
  };
}

function insertEvent(db, entry) {
  db.query(
    `
    insert into loop_events (loop_id, at, type, session_id, result, summary, details)
    values (?, ?, ?, ?, ?, ?, ?)
  `,
  ).run(
    entry.loopID,
    entry.at || nowIso(),
    entry.type,
    entry.sessionID || null,
    entry.result || null,
    entry.summary || null,
    entry.details ? JSON.stringify(entry.details) : null,
  );
}

function getLoop(db, loopID) {
  return db.query(`select * from loops where id = ?`).get(loopID) || null;
}

function listLoops(db, options = {}) {
  const clauses = [];
  const params = [];
  if (options.status === "active") clauses.push(`active = 1`);
  if (options.status === "inactive") clauses.push(`active = 0`);
  if (options.sessionID && options.sessionOnly) {
    clauses.push(`session_id = ?`);
    params.push(options.sessionID);
  }
  const where = clauses.length ? `where ${clauses.join(" and ")}` : "";
  const limit = Number(options.limit || 20);
  return db
    .query(
      `
    select * from loops
    ${where}
    order by active desc, pending desc, next_check_at asc, updated_at desc
    limit ?
  `,
    )
    .all(...params, limit);
}

function listEvents(db, options = {}) {
  const clauses = [];
  const params = [];
  if (options.loopID) {
    clauses.push(`loop_id = ?`);
    params.push(options.loopID);
  } else if (options.sessionID) {
    clauses.push(`session_id = ?`);
    params.push(options.sessionID);
  }
  const where = clauses.length ? `where ${clauses.join(" and ")}` : "";
  const limit = Number(options.limit || DEFAULT_HISTORY_LIMIT);
  return db
    .query(
      `
    select * from loop_events
    ${where}
    order by id desc
    limit ?
  `,
    )
    .all(...params, limit);
}

function formatLoop(row) {
  return `${row.id}: ${row.target} | ${row.last_result} | cycle ${row.cycle_count}/${row.max_cycles} | next ${row.next_check_at || "none"}${row.pending ? " | pending" : ""}`;
}

async function enqueueCycle(client, sessionID, loop, reason) {
  if (!client || !sessionID || !loop?.id) return false;
  const text = [
    "Run one observe-and-tune cycle now.",
    `Loop ID: ${loop.id}.`,
    `Target: ${loop.target || "unknown"}.`,
    `Success criteria: ${loop.success_criteria || "not specified"}.`,
    `Cycle: ${Number(loop.cycle_count) + 1} of ${loop.max_cycles}.`,
    `Trigger: ${reason}.`,
    `Reply with a short summary and include a final line exactly in this format: <observe-result loop_id="${loop.id}" status="healthy|degraded|blocked|failed" summary="..." />.`,
  ].join(" ");

  await client.session.prompt({
    path: { id: sessionID },
    body: { parts: [{ type: "text", text }], noReply: false },
  });
  return true;
}

function extractObserveResult(text) {
  if (!text) return null;
  const match = text.match(/<observe-result\b([^>]*)\/>/i);
  if (!match?.[1]) return null;
  const attrs = match[1];
  const attr = (name) =>
    attrs.match(new RegExp(`${name}="([^"]*)"`, "i"))?.[1]?.trim() || "";
  return {
    loopID: attr("loop_id") || attr("loop"),
    status: attr("status").toLowerCase(),
    summary: attr("summary"),
  };
}

function bindUnclaimedLoops(db, sessionID) {
  db.query(
    `update loops set session_id = ?, updated_at = ? where active = 1 and session_id is null`,
  ).run(sessionID, nowIso());
}

function activeLoopsForSession(db, sessionID) {
  return db
    .query(
      `
    select * from loops
    where active = 1 and session_id = ?
    order by pending desc, next_check_at asc, updated_at desc
  `,
    )
    .all(sessionID);
}

function pendingLoopForSession(db, sessionID) {
  return (
    db
      .query(
        `
    select * from loops
    where active = 1 and session_id = ? and pending = 1 and (lock_expires_at is null or lock_expires_at >= ?)
    order by updated_at desc
    limit 1
  `,
      )
      .get(sessionID, Date.now()) || null
  );
}

function nextDueLoop(db, sessionID) {
  db.query(
    `update loops set pending = 0, lock_expires_at = null where active = 1 and session_id = ? and pending = 1 and lock_expires_at is not null and lock_expires_at < ?`,
  ).run(sessionID, Date.now());
  return (
    db
      .query(
        `
    select * from loops
    where active = 1 and session_id = ? and pending = 0 and next_check_at is not null and next_check_at <= ?
    order by next_check_at asc, updated_at asc
    limit 1
  `,
      )
      .get(sessionID, nowIso()) || null
  );
}

function markLoopPending(db, loopID, trigger) {
  db.query(
    `
    update loops
    set pending = 1, lock_expires_at = ?, last_trigger = ?, updated_at = ?
    where id = ?
  `,
  ).run(Date.now() + LOCK_TTL_MS, trigger, nowIso(), loopID);
}

function resolveLoopForResult(db, sessionID, loopID) {
  if (loopID) {
    return (
      db
        .query(`select * from loops where id = ? and session_id = ?`)
        .get(loopID, sessionID) || null
    );
  }
  const pending = db
    .query(
      `select * from loops where active = 1 and session_id = ? and pending = 1 order by updated_at desc`,
    )
    .all(sessionID);
  return pending.length === 1 ? pending[0] : null;
}

function completeLoopCycle(db, loop, result) {
  const checkedAt = nowIso();
  const nextCycle = Number(loop.cycle_count) + 1;
  const shouldStop =
    result.status === "healthy" || nextCycle >= Number(loop.max_cycles);
  const nextCheckAt = shouldStop ? null : futureIso(Number(loop.interval_secs));
  db.query(
    `
    update loops
    set cycle_count = ?,
        pending = 0,
        lock_expires_at = null,
        last_result = ?,
        last_summary = ?,
        last_checked_at = ?,
        last_trigger = ?,
        active = ?,
        next_check_at = ?,
        completed_at = ?,
        updated_at = ?
    where id = ?
  `,
  ).run(
    nextCycle,
    result.status || "unknown",
    result.summary || "",
    checkedAt,
    "message.part.updated",
    shouldStop ? 0 : 1,
    nextCheckAt,
    shouldStop ? checkedAt : null,
    checkedAt,
    loop.id,
  );
  insertEvent(db, {
    loopID: loop.id,
    type: shouldStop ? "complete" : "cycle",
    sessionID: loop.session_id,
    result: result.status || "unknown",
    summary: result.summary || "",
    at: checkedAt,
    details: { cycle: nextCycle, nextCheckAt },
  });
}

async function injectResumeNote(client, sessionID, loops) {
  if (!client || !sessionID || !loops.length) return;
  const text = [
    "<observation-resume>",
    `Active loops: ${loops.length}.`,
    ...loops
      .slice(0, 5)
      .map(
        (loop) =>
          `Loop ${loop.id}: target=${loop.target}; cycle=${loop.cycle_count}/${loop.max_cycles}; result=${loop.last_result}; next=${loop.next_check_at || "none"}.`,
      ),
    "</observation-resume>",
  ]
    .filter(Boolean)
    .join(" ");
  await client.session.prompt({
    path: { id: sessionID },
    body: { noReply: true, parts: [{ type: "text", text }] },
  });
}

export const id = "observe-and-tune";

export const server = async ({ client, directory }) => {
  const { db, dbPath } = openObserveDb(directory);

  return {
    tool: {
      observe_start: tool({
        description: "Start bounded observation loop for current session.",
        args: {
          target: tool.schema.string().describe("URL, service, or target name"),
          interval_secs: tool.schema
            .number()
            .int()
            .min(5)
            .optional()
            .describe("Seconds between checks"),
          success_criteria: tool.schema
            .string()
            .optional()
            .describe("Health goal or exit criteria"),
          max_cycles: tool.schema
            .number()
            .int()
            .min(1)
            .optional()
            .describe("Hard stop after this many cycles"),
        },
        async execute(args, context) {
          const loopID = makeLoopID();
          const startedAt = nowIso();
          const target = args.target.trim();
          const interval = args.interval_secs ?? DEFAULT_INTERVAL;
          const maxCycles = args.max_cycles ?? DEFAULT_MAX_CYCLES;
          const criteria = args.success_criteria?.trim() ?? "";
          db.query(
            `
            insert into loops (
              id, session_id, target, interval_secs, max_cycles, success_criteria,
              active, cycle_count, pending, last_result, last_summary, last_trigger,
              started_at, next_check_at, created_at, updated_at
            ) values (?, ?, ?, ?, ?, ?, 1, 0, 0, 'scheduled', '', 'observe_start', ?, ?, ?, ?)
          `,
          ).run(
            loopID,
            context.sessionID,
            target,
            interval,
            maxCycles,
            criteria,
            startedAt,
            futureIso(interval),
            startedAt,
            startedAt,
          );
          insertEvent(db, {
            loopID,
            type: "start",
            sessionID: context.sessionID,
            summary: target,
            at: startedAt,
            details: { interval, maxCycles, criteria },
          });
          const loop = getLoop(db, loopID);
          context.metadata({
            title: "Observation started",
            metadata: { loop, dbPath },
          });
          return `Loop ${loopID} active for ${target}. Interval ${interval}s. Next ${loop?.next_check_at || "none"}.`;
        },
      }),
      observe_stop: tool({
        description: "Stop active observation loop.",
        args: {
          loop_id: tool.schema
            .string()
            .optional()
            .describe("Specific loop ID to stop"),
          all: tool.schema
            .boolean()
            .optional()
            .describe("Stop all active loops for this project"),
        },
        async execute(args, context) {
          const loops =
            args.all ?
              db
                .query(
                  `select * from loops where active = 1 order by started_at asc`,
                )
                .all()
            : args.loop_id ?
              db
                .query(`select * from loops where active = 1 and id = ?`)
                .all(args.loop_id)
            : db
                .query(
                  `select * from loops where active = 1 and session_id = ? order by started_at asc`,
                )
                .all(context.sessionID);
          if (!loops.length) {
            context.metadata({
              title: "Observation stopped",
              metadata: { count: 0, dbPath },
            });
            return "No matching active loops.";
          }
          const stoppedAt = nowIso();
          for (const loop of loops) {
            db.query(
              `
              update loops
              set active = 0, pending = 0, lock_expires_at = null, completed_at = ?, last_trigger = ?, updated_at = ?
              where id = ?
            `,
            ).run(
              stoppedAt,
              args.all ? "observe_stop:all" : "observe_stop",
              stoppedAt,
              loop.id,
            );
            insertEvent(db, {
              loopID: loop.id,
              type: "stop",
              sessionID: context.sessionID,
              result: loop.last_result,
              summary: loop.last_summary,
              at: stoppedAt,
            });
          }
          context.metadata({
            title: "Observation stopped",
            metadata: { count: loops.length, dbPath },
          });
          return `Stopped ${loops.length} loop(s).`;
        },
      }),
      observe_status: tool({
        description: "Show current observation status.",
        args: {
          loop_id: tool.schema.string().optional().describe("Specific loop ID"),
        },
        async execute(args, context) {
          if (args.loop_id) {
            const loop = getLoop(db, args.loop_id);
            if (!loop) throw new Error(`Unknown observe loop: ${args.loop_id}`);
            const events = listEvents(db, {
              loopID: args.loop_id,
              limit: 5,
            }).reverse();
            context.metadata({
              title: "Observation status",
              metadata: { loop, dbPath },
            });
            return [
              formatLoop(loop),
              events.length ?
                `Recent: ${events.map((event) => `${event.at}: ${event.type}${event.result ? ` ${event.result}` : ""}${event.summary ? ` ${event.summary}` : ""}`).join(" | ")}`
              : null,
            ]
              .filter(Boolean)
              .join("\n");
          }
          const loops = activeLoopsForSession(db, context.sessionID);
          context.metadata({
            title: "Observation status",
            metadata: { count: loops.length, dbPath },
          });
          return loops.length ?
              loops.map(formatLoop).join("\n")
            : "Observation idle.";
        },
      }),
      observe_list: tool({
        description: "List observe loops for this project.",
        args: {
          status: tool.schema
            .enum(["active", "inactive", "all"])
            .optional()
            .describe("Loop status filter"),
          session_only: tool.schema
            .boolean()
            .optional()
            .describe("Restrict to current session"),
          limit: tool.schema
            .number()
            .int()
            .min(1)
            .max(100)
            .optional()
            .describe("Max loops to return"),
        },
        async execute(args, context) {
          const loops = listLoops(db, {
            status: args.status || "active",
            sessionOnly: Boolean(args.session_only),
            sessionID: context.sessionID,
            limit: args.limit || 20,
          });
          context.metadata({
            title: "Observation list",
            metadata: { count: loops.length, dbPath },
          });
          return loops.length ?
              loops.map(formatLoop).join("\n")
            : "No matching observe loops.";
        },
      }),
      observe_history: tool({
        description: "Show recent observe history for a loop or session.",
        args: {
          loop_id: tool.schema.string().optional().describe("Specific loop ID"),
          limit: tool.schema
            .number()
            .int()
            .min(1)
            .max(100)
            .optional()
            .describe("Max events to return"),
        },
        async execute(args, context) {
          const events = listEvents(db, {
            loopID: args.loop_id,
            sessionID: args.loop_id ? null : context.sessionID,
            limit: args.limit || DEFAULT_HISTORY_LIMIT,
          }).reverse();
          context.metadata({
            title: "Observation history",
            metadata: {
              count: events.length,
              dbPath,
              loopID: args.loop_id || null,
            },
          });
          return events.length ?
              events
                .map(
                  (event) =>
                    `${event.at}: loop=${event.loop_id} ${event.type}${event.result ? ` ${event.result}` : ""}${event.summary ? ` ${event.summary}` : ""}`,
                )
                .join("\n")
            : "No observe history found.";
        },
      }),
      observe_prune: tool({
        description: "Prune old observe loops/events and show retained counts.",
        args: {},
        async execute(_args, context) {
          const result = pruneObserveDbWithCounts(db);
          context.metadata({
            title: "Observation prune",
            metadata: { ...result, dbPath },
          });
          return `Removed loops=${result.loopsRemoved} events=${result.eventsRemoved}. Remaining loops=${result.remainingLoops} events=${result.remainingEvents}.`;
        },
      }),
    },

    event: async ({ event }) => {
      if (event.type === "session.created") {
        const newSessionID =
          event.properties.sessionID || event.properties.info?.id || null;
        if (!newSessionID) return;
        bindUnclaimedLoops(db, newSessionID);
        const loops = activeLoopsForSession(db, newSessionID);
        try {
          await injectResumeNote(client, newSessionID, loops);
        } catch {
          // best effort only
        }
        return;
      }

      if (event.type === "session.deleted") {
        const sessionID = event.properties.info?.id;
        if (sessionID) {
          const loops = activeLoopsForSession(db, sessionID);
          const detachedAt = nowIso();
          for (const loop of loops) {
            db.query(
              `
              update loops
              set session_id = null, pending = 0, lock_expires_at = null, last_trigger = ?, updated_at = ?
              where id = ?
            `,
            ).run("session.deleted", detachedAt, loop.id);
            insertEvent(db, {
              loopID: loop.id,
              type: "session-detached",
              sessionID,
              result: loop.last_result,
              summary: loop.last_summary,
              at: detachedAt,
            });
          }
        }
        return;
      }

      if (event.type === "session.compacted") {
        const loops = activeLoopsForSession(db, event.properties.sessionID);
        if (!loops.length) return;
        try {
          await injectResumeNote(client, event.properties.sessionID, loops);
        } catch {
          // best effort only
        }
        return;
      }

      if (event.type === "session.idle") {
        pruneObserveDb(db);
        if (pendingLoopForSession(db, event.properties.sessionID)) return;
        const loop = nextDueLoop(db, event.properties.sessionID);
        if (!loop) return;
        markLoopPending(db, loop.id, "session.idle");
        insertEvent(db, {
          loopID: loop.id,
          type: "enqueue",
          sessionID: event.properties.sessionID,
          summary: loop.target,
          details: { cycle: Number(loop.cycle_count) + 1 },
        });
        try {
          await enqueueCycle(
            client,
            event.properties.sessionID,
            { ...loop, pending: 1 },
            "session.idle",
          );
        } catch {
          const failedAt = nowIso();
          db.query(
            `
            update loops
            set pending = 0, lock_expires_at = null, last_result = ?, last_summary = ?, last_trigger = ?, next_check_at = ?, updated_at = ?
            where id = ?
          `,
          ).run(
            "failed",
            "Could not enqueue observe cycle",
            "session.idle:enqueue-failed",
            futureIso(Number(loop.interval_secs)),
            failedAt,
            loop.id,
          );
          insertEvent(db, {
            loopID: loop.id,
            type: "enqueue-failed",
            sessionID: event.properties.sessionID,
            result: "failed",
            summary: "Could not enqueue observe cycle",
            at: failedAt,
          });
        }
        return;
      }

      if (event.type === "message.part.updated") {
        const part = event.properties.part;
        if (part.type !== "text" || part.synthetic) return;
        const result = extractObserveResult(part.text);
        if (!result?.status) return;
        const loop = resolveLoopForResult(
          db,
          event.properties.sessionID,
          result.loopID,
        );
        if (!loop) return;
        completeLoopCycle(db, loop, result);
      }
    },

    "experimental.chat.system.transform": async (input, output) => {
      const loops = activeLoopsForSession(db, input.sessionID);
      if (!loops.length) return;
      output.system.push(
        [
          "<observation-state>",
          `db_path=${dbPath}`,
          `active_loops=${loops.length}`,
          ...loops
            .slice(0, 5)
            .map(
              (loop) =>
                `loop=${loop.id} target=${loop.target} cycle=${loop.cycle_count}/${loop.max_cycles} result=${loop.last_result} pending=${Boolean(loop.pending)} next=${loop.next_check_at || "none"}`,
            ),
          "</observation-state>",
        ].join("\n"),
      );
    },
  };
};

export default { id, server };
