# TASK

Merge the following branches into the current branch:

{{BRANCHES}}

For each branch:

1. Run `git merge <branch> --no-edit`
2. If there are merge conflicts, resolve them intelligently by reading both sides and choosing the correct resolution
3. After resolving conflicts, run `npm run typecheck` and `npm run test` to verify everything works
4. If tests fail, fix the issues before proceeding to the next branch

After all branches are merged, make a single commit summarizing the merge.

# ISSUE CLOSING

Do not close GitHub issues yourself. The Sandcastle orchestrator closes each
issue after this merge run returns and verifies that the matching branch is an
ancestor of `HEAD`.

Here are the issues for merge context:

{{ISSUES}}

Once you've merged everything you can, output <promise>COMPLETE</promise>.
