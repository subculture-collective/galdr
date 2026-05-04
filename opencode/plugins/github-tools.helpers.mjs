import fs from "node:fs";

export function shellEscape(value) {
  return `'${String(value).replace(/'/g, `'\\''`)}'`;
}

export async function runShell($, command) {
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

export async function resolvePrSelector($, prNumber) {
  if (prNumber) return String(prNumber);
  const current = await runShell(
    $,
    `gh pr view --json number --jq '.number' 2>/dev/null`,
  );
  if (/^\d+$/.test(current)) return current;
  const branch = await runShell($, `git rev-parse --abbrev-ref HEAD`);
  if (!branch) return null;
  const branchPr = await runShell(
    $,
    `gh pr list --head ${shellEscape(branch)} --json number --jq '.[0].number' 2>/dev/null`,
  );
  return /^\d+$/.test(branchPr) ? branchPr : null;
}

export function buildIncludeArg(raw) {
  if (!raw) return "";
  const value = String(raw).trim();
  if (!value) return "";
  if (value.includes("*") || value.includes("{") || value.includes("/")) {
    return ` -g ${shellEscape(value)}`;
  }
  if (value.includes(",")) {
    return value
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean)
      .map((part) => ` -g ${shellEscape(`*.${part.replace(/^\./, "")}`)}`)
      .join("");
  }
  if (value === "test") {
    return ` -g ${shellEscape("*.test.*")} -g ${shellEscape("*.spec.*")}`;
  }
  return ` -g ${shellEscape(`*.${value.replace(/^\./, "")}`)}`;
}

export function repoFlag(repo) {
  return repo ? ` --repo ${shellEscape(repo)}` : "";
}

export function optionalFlag(flag, value) {
  return value !== undefined && value !== null && String(value).trim() !== "" ?
      ` ${flag} ${shellEscape(String(value))}`
    : "";
}

export function buildTree(text, depth) {
  const seen = new Set();
  const lines = [];
  for (const raw of text.split(/\r?\n/)) {
    const value = raw.trim();
    if (!value) continue;
    const parts = value.split("/").slice(0, depth);
    const cur = [];
    for (const part of parts) {
      cur.push(part);
      const key = cur.join("/");
      if (seen.has(key)) continue;
      seen.add(key);
      lines.push(`${"  ".repeat(cur.length - 1)}${part}`);
    }
  }
  return lines.join("\n");
}

export async function collectSuiteStatus($, repo) {
  const auth = await runShell($, `gh auth status --active 2>/dev/null`);
  const gitRoot = await runShell(
    $,
    `git rev-parse --show-toplevel 2>/dev/null`,
  );
  const branch =
    gitRoot ? await runShell($, `git rev-parse --abbrev-ref HEAD`) : "";
  const remote =
    gitRoot ? await runShell($, `git remote get-url origin 2>/dev/null`) : "";
  const repoInfo = await runShell(
    $,
    `gh repo view${repoFlag(repo)} --json name,owner,visibility,defaultBranchRef --jq '.owner.login + "/" + .name + " | " + .visibility + " | default=" + .defaultBranchRef.name' 2>/dev/null`,
  );
  const prSelector = gitRoot ? await resolvePrSelector($, null) : null;
  const prInfo =
    prSelector ?
      await runShell(
        $,
        `gh pr view ${prSelector} --json number,title,state,isDraft,reviewDecision --jq '"#" + (.number|tostring) + " " + .title + " | " + .state + " | draft=" + (.isDraft|tostring) + " | review=" + (.reviewDecision // "none")'`,
      )
    : "";
  const runs = await runShell(
    $,
    `gh run list --limit 10${repoFlag(repo)} --json databaseId,workflowName,status,conclusion,headBranch --jq '.[] | select((.status != "completed") or (.conclusion != null and .conclusion != "success")) | "#" + (.databaseId|tostring) + " " + (.workflowName // "workflow") + " | branch=" + (.headBranch // "") + " | status=" + .status + " | conclusion=" + (.conclusion // "none")' 2>/dev/null`,
  );
  const reviewQueue = await runShell(
    $,
    `gh pr list --state open${repoFlag(repo)} --json number,title,isDraft,reviewDecision,updatedAt,author --jq '.[] | select((.isDraft|not) and (.reviewDecision != "APPROVED")) | "#" + (.number|tostring) + " " + .title + " | @" + .author.login + " | review=" + (.reviewDecision // "none") + " | updated=" + .updatedAt' 2>/dev/null`,
  );
  return {
    auth,
    gitRoot,
    branch,
    remote,
    repoInfo,
    prSelector,
    prInfo,
    runs,
    reviewQueue,
  };
}

export function resolveSecretInput(args) {
  if (args.value_env) {
    const value = process.env[String(args.value_env)];
    if (value === undefined) {
      throw new Error(`Environment variable not set: ${args.value_env}`);
    }
    return value;
  }
  if (args.value_file) {
    return fs.readFileSync(String(args.value_file), "utf8");
  }
  if (args.value !== undefined) return String(args.value);
  throw new Error("Provide one of: value, value_env, value_file");
}
