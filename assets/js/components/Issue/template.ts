import type { IssueData, Sections, GitHubContent, Template, HelpType } from "./types";

// Constants
const PLACEHOLDER = "⚠️  RETURN TO EVCC TAB → COPY STEP 2 → PASTE HERE";
const MAX_BODY_LENGTH = 8000;

function toString(sections: Template): string {
  return sections
    .map((section) => (Array.isArray(section) ? section.join("\n") : section))
    .join("\n\n");
}

export function generateGitHubContent(issue: IssueData, sections: Sections): GitHubContent {
  const additional = generateAdditional(sections);

  // First attempt: generate body with summary details included
  let body = generateBody(issue, additional);

  // Check if it fits within the limit
  if (encodeURIComponent(issue.title + body).length <= MAX_BODY_LENGTH) {
    return { body };
  }

  // If too long, generate body with placeholder and return additional separately
  body = generateBody(issue, PLACEHOLDER);
  return { body, additional };
}

function generateBody(issue: IssueData, additional: string): string {
  const sections: Template = [
    "## Description",
    issue.description,
    "## Steps to Reproduce",
    issue.steps,
    additional,
    "## Version",
    issue.version,
  ];

  return toString(sections);
}

function generateAdditional(sections: Sections): string {
  const result: Template = [];

  if (sections.yamlConfig.included) {
    result.push("## Configuration (YAML)");
    result.push(["```yaml", sections.yamlConfig.content, "```"]);
  }

  if (sections.uiConfig.included) {
    result.push("## Configuration (UI)");
    result.push(["```json5", sections.uiConfig.content, "```"]);
  }

  if (sections.state.included) {
    result.push("## System State");
    result.push(["```json5", sections.state.content, "```"]);
  }

  if (sections.logs.included) {
    result.push("## Logs");
    result.push(["```", sections.logs.content, "```"]);
  }

  return toString(result);
}

// Generates GitHub URL for issues or discussions
export function generateGitHubUrl(type: HelpType, title: string, body: string): string {
  const baseUrl =
    type === "discussion"
      ? "https://github.com/evcc-io/evcc/discussions/new?category=need-help&"
      : "https://github.com/evcc-io/evcc/issues/new?";

  return `${baseUrl}title=${encodeURIComponent(title)}&body=${encodeURIComponent(body)}`;
}
