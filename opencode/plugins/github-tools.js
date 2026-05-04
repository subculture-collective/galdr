import { tool } from "@opencode-ai/plugin";

import * as helpers from "./github-tools.helpers.mjs";
import { createRepoTools } from "./github-tools.repo.mjs";
import { createIssuePrTools } from "./github-tools.issues-prs.mjs";
import { createStatusTools } from "./github-tools.status.mjs";

export const id = "github-tools";

export const server = async ({ $ }) => {
  return {
    tool: {
      ...createRepoTools({ tool, $, helpers }),
      ...createIssuePrTools({ tool, $, helpers }),
      ...createStatusTools({ tool, $, helpers }),
    },
  };
};

export default { id, server };
