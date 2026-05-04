export function createRepoTools({ tool, $, helpers }) {
  const { shellEscape, runShell, buildIncludeArg, buildTree } = helpers;

  return {
    gh_tree: tool({
      description: "Show repo tree for current checkout.",
      args: {
        path: tool.schema.string().optional().describe("Optional subdirectory"),
        depth: tool.schema
          .number()
          .int()
          .min(1)
          .max(8)
          .optional()
          .describe("Max tree depth"),
      },
      async execute(args, context) {
        const depth = args.depth ?? 3;
        const target = args.path ? ` ${shellEscape(args.path)}` : "";
        const files = await runShell(
          $,
          `rg --files --hidden -g '!.git'${target}`,
        );
        const output = buildTree(files, depth);
        context.metadata({
          title: "GitHub tree",
          metadata: { path: args.path || ".", depth },
        });
        return output || "No tracked files found.";
      },
    }),
    gh_search: tool({
      description: "Search tracked repo files with ripgrep.",
      args: {
        pattern: tool.schema.string().describe("Search pattern"),
        type: tool.schema
          .string()
          .optional()
          .describe(
            "Optional file include like ts, md, test, *.tsx, src/**/*.ts",
          ),
      },
      async execute(args, context) {
        const include = buildIncludeArg(args.type);
        const output = await runShell(
          $,
          `rg -n --hidden --glob '!.git'${include} ${shellEscape(args.pattern)}`,
        );
        context.metadata({
          title: "GitHub search",
          metadata: { pattern: args.pattern, type: args.type || null },
        });
        return output || "No matches found.";
      },
    }),
    gh_branch_list: tool({
      description: "List remote branches in the current repository.",
      args: {},
      async execute(_args, context) {
        const output = await runShell(
          $,
          `git for-each-ref --format='%(refname:short)' refs/remotes/origin | sed 's#^origin/##' | sort -u`,
        );
        context.metadata({ title: "Branch list" });
        return output || "No remote branches found.";
      },
    }),
  };
}
