import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { pathToFileURL } from "node:url";

export const OPENCODE_CONFIG_DIR = path.join(
  os.homedir(),
  ".config",
  "opencode",
);
export const OPENCODE_PLUGIN_DIR = path.join(OPENCODE_CONFIG_DIR, "plugins");
export const OPENCODE_COMMAND_DIR = path.join(OPENCODE_CONFIG_DIR, "commands");
export const OPENCODE_CACHE_DIR = path.join(
  os.homedir(),
  ".cache",
  "opencode",
  "packages",
);

export function ensureDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

export function readFileSafe(filePath, fallback = "") {
  try {
    return fs.readFileSync(filePath, "utf8");
  } catch {
    return fallback;
  }
}

export function loadJson(filePath, fallback) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return fallback;
  }
}

export function saveJson(filePath, value) {
  ensureDir(filePath);
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

export function hashString(value) {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i += 1) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(16);
}

export function projectKey(directory) {
  return hashString(path.resolve(directory));
}

export function observeProjectDbPath(directory) {
  return path.join(
    OPENCODE_CONFIG_DIR,
    "observe",
    `${projectKey(directory)}.db`,
  );
}

export function findBinaryOnPath(name) {
  const pathEnv = process.env.PATH || "";
  for (const dir of pathEnv.split(":")) {
    if (!dir) continue;
    const full = path.join(dir, name);
    try {
      fs.accessSync(full, fs.constants.X_OK);
      return full;
    } catch {
      // ignore
    }
  }
  return null;
}

export function listLocalPluginFiles() {
  try {
    return fs
      .readdirSync(OPENCODE_PLUGIN_DIR)
      .filter((name) => name.endsWith(".js") && !name.startsWith("_"))
      .sort();
  } catch {
    return [];
  }
}

export function listCommandFiles() {
  try {
    return fs
      .readdirSync(OPENCODE_COMMAND_DIR)
      .filter((name) => name.endsWith(".md"))
      .sort();
  } catch {
    return [];
  }
}

export function resolveCachedPackageEntry(packageName, relativeFile) {
  let entries = [];
  try {
    entries = fs.readdirSync(OPENCODE_CACHE_DIR, { withFileTypes: true });
  } catch {
    return null;
  }

  const matches = entries
    .filter(
      (entry) =>
        entry.isDirectory() && entry.name.startsWith(`${packageName}@`),
    )
    .map((entry) => {
      const full = path.join(
        OPENCODE_CACHE_DIR,
        entry.name,
        "node_modules",
        packageName,
        relativeFile,
      );
      try {
        const stat = fs.statSync(full);
        return { full, mtimeMs: stat.mtimeMs };
      } catch {
        return null;
      }
    })
    .filter(Boolean)
    .sort((a, b) => b.mtimeMs - a.mtimeMs || b.full.localeCompare(a.full));

  return matches[0]?.full || null;
}

export async function importCachedPackageModule(packageName, relativeFile) {
  const full = resolveCachedPackageEntry(packageName, relativeFile);
  if (!full) return null;
  try {
    return await import(pathToFileURL(full).href);
  } catch {
    return null;
  }
}

export function listFilesRecursive(root, opts = {}) {
  const out = [];
  const ignore = new Set(
    opts.ignore || [
      "node_modules",
      ".git",
      "dist",
      "build",
      ".next",
      ".turbo",
      "coverage",
    ],
  );
  function walk(current) {
    let entries = [];
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      if (ignore.has(entry.name)) continue;
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      out.push(full);
    }
  }
  walk(root);
  return out;
}
