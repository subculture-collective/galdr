export function createIssuePrTools({ tool, $, helpers }) {
  const { shellEscape, runShell, repoFlag, optionalFlag, resolvePrSelector } =
    helpers;

  return {
    gh_issue_create: tool({
      description: "Create a GitHub issue with labels and assignees.",
      args: {
        title: tool.schema.string().describe("Issue title"),
        body: tool.schema.string().optional().describe("Issue body"),
        labels: tool.schema
          .string()
          .optional()
          .describe("Comma-separated labels"),
        assignees: tool.schema
          .string()
          .optional()
          .describe("Comma-separated assignees or @me"),
        milestone: tool.schema
          .string()
          .optional()
          .describe("Optional milestone title"),
        repo: tool.schema.string().optional().describe("Optional owner/repo"),
      },
      async execute(args, context) {
        const command = [
          `gh issue create --title ${shellEscape(args.title)}`,
          optionalFlag("--body", args.body),
          optionalFlag("--label", args.labels),
          optionalFlag("--assignee", args.assignees),
          optionalFlag("--milestone", args.milestone),
          repoFlag(args.repo),
        ].join("");
        const output = await runShell($, command);
        context.metadata({
          title: "Issue create",
          metadata: { title: args.title, repo: args.repo || "current" },
        });
        return output || `Created issue ${args.title}.`;
      },
    }),
    gh_issue_update: tool({
      description: "Update issue title/body/state/labels/assignees.",
      args: {
        issue_number: tool.schema
          .number()
          .int()
          .min(1)
          .describe("Issue number"),
        title: tool.schema.string().optional().describe("Updated title"),
        body: tool.schema.string().optional().describe("Updated body"),
        state: tool.schema
          .enum(["open", "closed"])
          .optional()
          .describe("Open or close issue"),
        add_labels: tool.schema
          .string()
          .optional()
          .describe("Comma-separated labels to add"),
        remove_labels: tool.schema
          .string()
          .optional()
          .describe("Comma-separated labels to remove"),
        add_assignees: tool.schema
          .string()
          .optional()
          .describe("Comma-separated assignees to add"),
        remove_assignees: tool.schema
          .string()
          .optional()
          .describe("Comma-separated assignees to remove"),
        milestone: tool.schema
          .string()
          .optional()
          .describe("Optional milestone title"),
        repo: tool.schema.string().optional().describe("Optional owner/repo"),
      },
      async execute(args, context) {
        const editCmd = [
          `gh issue edit ${args.issue_number}`,
          optionalFlag("--title", args.title),
          optionalFlag("--body", args.body),
          optionalFlag("--add-label", args.add_labels),
          optionalFlag("--remove-label", args.remove_labels),
          optionalFlag("--add-assignee", args.add_assignees),
          optionalFlag("--remove-assignee", args.remove_assignees),
          optionalFlag("--milestone", args.milestone),
          repoFlag(args.repo),
        ].join("");
        const output = await runShell($, editCmd);
        const stateCmd =
          args.state === "closed" ?
            await runShell(
              $,
              `gh issue close ${args.issue_number}${repoFlag(args.repo)}`,
            )
          : args.state === "open" ?
            await runShell(
              $,
              `gh issue reopen ${args.issue_number}${repoFlag(args.repo)}`,
            )
          : "";
        context.metadata({
          title: "Issue update",
          metadata: { issue: args.issue_number, repo: args.repo || "current" },
        });
        return (
          [output, stateCmd].filter(Boolean).join("\n") ||
          `Updated issue #${args.issue_number}.`
        );
      },
    }),
    gh_issue_comment: tool({
      description: "Add a comment to an issue.",
      args: {
        issue_number: tool.schema
          .number()
          .int()
          .min(1)
          .describe("Issue number"),
        body: tool.schema.string().describe("Comment body"),
        repo: tool.schema.string().optional().describe("Optional owner/repo"),
      },
      async execute(args, context) {
        const output = await runShell(
          $,
          `gh issue comment ${args.issue_number} --body ${shellEscape(args.body)}${repoFlag(args.repo)}`,
        );
        context.metadata({
          title: "Issue comment",
          metadata: { issue: args.issue_number, repo: args.repo || "current" },
        });
        return output || `Commented on issue #${args.issue_number}.`;
      },
    }),
    gh_issue_list: tool({
      description: "List GitHub issues with optional label and state.",
      args: {
        label: tool.schema.string().optional().describe("Label filter"),
        state: tool.schema.string().optional().describe("open, closed, all"),
      },
      async execute(args, context) {
        const state = args.state || "open";
        const label = args.label ? ` --label ${shellEscape(args.label)}` : "";
        const output = await runShell(
          $,
          `gh issue list --state ${shellEscape(state)}${label} --json number,title,labels,assignees --jq '.[] | "#" + (.number|tostring) + " " + .title'`,
        );
        context.metadata({
          title: "Issue list",
          metadata: { state, label: args.label || null },
        });
        return output || `No ${state} issues.`;
      },
    }),
    gh_pr_diff: tool({
      description: "Get diff for a GitHub pull request.",
      args: {
        pr_number: tool.schema
          .number()
          .int()
          .min(1)
          .describe("Pull request number"),
      },
      async execute(args, context) {
        const output = await runShell($, `gh pr diff ${args.pr_number}`);
        context.metadata({
          title: "PR diff",
          metadata: { pr: args.pr_number },
        });
        return output || `No diff output for PR ${args.pr_number}.`;
      },
    }),
    gh_pr_comments: tool({
      description: "Get review comments grouped for a PR.",
      args: {
        pr_number: tool.schema
          .number()
          .int()
          .min(1)
          .describe("Pull request number"),
      },
      async execute(args, context) {
        const output = await runShell(
          $,
          `gh pr view ${args.pr_number} --comments`,
        );
        context.metadata({
          title: "PR comments",
          metadata: { pr: args.pr_number },
        });
        return output || `No comments for PR ${args.pr_number}.`;
      },
    }),
    gh_pr_comment: tool({
      description: "Add a general comment to a pull request.",
      args: {
        pr_number: tool.schema
          .number()
          .int()
          .min(1)
          .describe("Pull request number"),
        body: tool.schema.string().describe("Comment body"),
        repo: tool.schema.string().optional().describe("Optional owner/repo"),
      },
      async execute(args, context) {
        const output = await runShell(
          $,
          `gh pr comment ${args.pr_number} --body ${shellEscape(args.body)}${repoFlag(args.repo)}`,
        );
        context.metadata({
          title: "PR comment",
          metadata: { pr: args.pr_number, repo: args.repo || "current" },
        });
        return output || `Commented on PR #${args.pr_number}.`;
      },
    }),
    gh_pr_review: tool({
      description: "Approve, comment on, or request changes on a PR.",
      args: {
        pr_number: tool.schema
          .number()
          .int()
          .min(1)
          .describe("Pull request number"),
        action: tool.schema
          .enum(["approve", "comment", "request-changes"])
          .describe("Review action"),
        body: tool.schema.string().optional().describe("Optional review body"),
        repo: tool.schema.string().optional().describe("Optional owner/repo"),
      },
      async execute(args, context) {
        const actionFlag =
          args.action === "approve" ? "--approve"
          : args.action === "request-changes" ? "--request-changes"
          : "--comment";
        const output = await runShell(
          $,
          `gh pr review ${args.pr_number} ${actionFlag}${optionalFlag("--body", args.body)}${repoFlag(args.repo)}`,
        );
        context.metadata({
          title: "PR review",
          metadata: {
            pr: args.pr_number,
            action: args.action,
            repo: args.repo || "current",
          },
        });
        return output || `${args.action} sent for PR #${args.pr_number}.`;
      },
    }),
    gh_pr_context: tool({
      description:
        "Show current branch PR context including checks, files, and comments summary.",
      args: {
        pr_number: tool.schema
          .number()
          .int()
          .min(1)
          .optional()
          .describe(
            "Optional pull request number; defaults to current branch PR",
          ),
      },
      async execute(args, context) {
        const selector = await resolvePrSelector($, args.pr_number);
        if (!selector) {
          context.metadata({ title: "PR context", metadata: { pr: null } });
          return "No PR found for current branch.";
        }
        const pr = await runShell(
          $,
          `gh pr view ${selector} --json number,title,state,isDraft,headRefName,baseRefName,author,reviewDecision`,
        );
        const checks = await runShell($, `gh pr checks ${selector}`);
        const files = await runShell(
          $,
          `gh pr view ${selector} --json files --jq '.files[].path'`,
        );
        const comments = await runShell($, `gh pr view ${selector} --comments`);
        context.metadata({ title: "PR context", metadata: { pr: selector } });
        return [
          "PR:",
          pr || "none",
          "",
          "Checks:",
          checks || "none",
          "",
          "Files:",
          files || "none",
          "",
          "Comments:",
          comments || "none",
        ].join("\n");
      },
    }),
    gh_review_queue: tool({
      description: "List open pull requests likely needing review attention.",
      args: {},
      async execute(_args, context) {
        const output = await runShell(
          $,
          `gh pr list --state open --json number,title,isDraft,reviewDecision,statusCheckRollup,updatedAt,author --jq '.[] | select((.isDraft|not) and (.reviewDecision != "APPROVED")) | "#" + (.number|tostring) + " " + .title + " | @" + .author.login + " | review=" + (.reviewDecision // "none") + " | updated=" + .updatedAt'`,
        );
        context.metadata({ title: "Review queue" });
        return output || "No open PRs needing review attention.";
      },
    }),
    gh_changed_files: tool({
      description: "List files changed against main or master.",
      args: {},
      async execute(_args, context) {
        const output = await runShell(
          $,
          `base=$(git rev-parse --verify main >/dev/null 2>&1 && printf main || printf master); git diff --name-only "$base"...HEAD`,
        );
        context.metadata({ title: "Changed files" });
        return output || "No changed files against main/master.";
      },
    }),
    gh_ci_status: tool({
      description: "Show CI status for current branch or ref.",
      args: {
        ref: tool.schema.string().optional().describe("Git ref or branch"),
      },
      async execute(args, context) {
        const ref = args.ref ? shellEscape(args.ref) : "";
        const output = await runShell($, `gh pr checks ${ref || ""}`);
        context.metadata({
          title: "CI status",
          metadata: { ref: args.ref || "current" },
        });
        return output || "No CI status available.";
      },
    }),
  };
}
