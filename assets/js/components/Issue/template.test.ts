import { describe, it, expect } from "vitest";
import { generateGitHubContent } from "./template";
import type { IssueData, Sections } from "./types";

describe("Issue Utils", () => {
  const mockIssueData: IssueData = {
    title: "Test Issue",
    description: "This is a test description",
    steps: "1. Do something\n2. See error",
    version: "v1.0.0",
  };

  const mockSections: Sections = {
    yamlConfig: { included: true, content: "key: value\nother: test" },
    uiConfig: { included: true, content: '{"setting": "value"}' },
    state: { included: false, content: "" },
    logs: { included: true, content: "2023-01-01 ERROR: Something went wrong" },
  };

  describe("generateGitHubContent", () => {
    it("generates body with additional content", () => {
      const result = generateGitHubContent(mockIssueData, mockSections);

      expect(result.body).toBe(`## Description

This is a test description

## Steps to Reproduce

1. Do something
2. See error

## Configuration (YAML)

\`\`\`yaml
key: value
other: test
\`\`\`

## Configuration (UI)

\`\`\`json5
{"setting": "value"}
\`\`\`

## Logs

\`\`\`
2023-01-01 ERROR: Something went wrong
\`\`\`

## Version

v1.0.0`);
      expect(result.additional).toBeUndefined();
    });

    it("uses placeholder when exceeding limit", () => {
      const longContent = "x".repeat(8000);
      const longSections: Sections = {
        ...mockSections,
        yamlConfig: {
          included: true,
          content: longContent,
        },
      };

      const result = generateGitHubContent(mockIssueData, longSections);

      expect(result.body).toContain("⚠️  RETURN TO EVCC TAB → COPY STEP 2 → PASTE HERE");
      expect(result.body).not.toContain("## Configuration (YAML)");
      expect(result.additional).toBeDefined();
      expect(result.additional).toContain(longContent);
    });

    it("handles empty steps", () => {
      const issueWithoutSteps: IssueData = {
        ...mockIssueData,
        steps: "",
      };

      const result = generateGitHubContent(issueWithoutSteps, mockSections);

      expect(result.body).toContain("## Steps to Reproduce");
      expect(result.body).toContain("## Steps to Reproduce\n\n\n\n## Configuration");
      expect(result.body).toContain("## Configuration (YAML)");
    });

    it("includes only enabled sections", () => {
      const selectiveSections: Sections = {
        yamlConfig: { included: true, content: "yaml: content" },
        uiConfig: { included: false, content: "ui: content" },
        state: { included: true, content: "state: content" },
        logs: { included: false, content: "log: content" },
      };

      const result = generateGitHubContent(mockIssueData, selectiveSections);

      expect(result.body).toContain("## Configuration (YAML)");
      expect(result.body).toContain("## System State");
      expect(result.body).not.toContain("## Configuration (UI)");
      expect(result.body).not.toContain("## Logs");
    });

    it("handles all sections disabled", () => {
      const emptySections: Sections = {
        yamlConfig: { included: false, content: "" },
        uiConfig: { included: false, content: "" },
        state: { included: false, content: "" },
        logs: { included: false, content: "" },
      };

      const result = generateGitHubContent(mockIssueData, emptySections);

      expect(result.body).not.toContain("## Configuration");
      expect(result.body).not.toContain("## System State");
      expect(result.body).not.toContain("## Logs");
    });
  });
});
