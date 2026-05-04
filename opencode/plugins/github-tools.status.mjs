export function createStatusTools({ tool, $, helpers }) {
  const { collectSuiteStatus, repoFlag } = helpers;

  return {
    gh_suite_status: tool({
      description: "Summarize GitHub auth, repo, PR, runs, and review queue.",
      args: {
        repo: tool.schema
          .string()
          .optional()
          .describe("Optional owner/repo override"),
      },
      async execute(args, context) {
        const status = await collectSuiteStatus($, args.repo);
        context.metadata({ title: "GH suite status", metadata: status });
        return [
          `Auth: ${status.auth ? "ok" : "missing"}`,
          status.repoInfo ? `Repo: ${status.repoInfo}` : "Repo: none",
          status.branch ? `Branch: ${status.branch}` : null,
          status.prInfo ? `PR: ${status.prInfo}` : "PR: none",
          "",
          "Failing/active runs:",
          status.runs || "none",
          "",
          "Review queue:",
          status.reviewQueue || "none",
        ]
          .filter(Boolean)
          .join("\n");
      },
    }),
    gh_doctor: tool({
      description: "Diagnose GitHub CLI/auth/repo context problems.",
      args: {
        repo: tool.schema
          .string()
          .optional()
          .describe("Optional owner/repo override"),
      },
      async execute(args, context) {
        const status = await collectSuiteStatus($, args.repo);
        const issues = [];
        if (!status.auth) issues.push("gh auth missing or invalid");
        if (!status.gitRoot) issues.push("not inside git repo");
        if (!status.repoInfo) {
          issues.push("gh repo view failed; current repo may be unknown to gh");
        }
        if (status.gitRoot && !status.remote) {
          issues.push("git remote origin missing");
        }
        context.metadata({ title: "GH doctor", metadata: status });
        return issues.length ?
            [
              "Issues:",
              ...issues.map((issue) => `- ${issue}`),
              "",
              status.auth || "",
              status.repoInfo || "",
            ].join("\n")
          : "No obvious GitHub CLI issues.";
      },
    }),
  };
}
