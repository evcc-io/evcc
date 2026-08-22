import { describe, it, expect } from "vite-plus/test";
import {
  generateGitHubContent,
  generateMailtoUrl,
  generateDebugFile,
  MAX_MAIL_TITLE_LENGTH,
  MAX_MAIL_DESCRIPTION_LENGTH,
} from "./template";
import type { IssueData, Sections } from "./types";

describe("Issue Utils", () => {
  const mockIssueData: IssueData = {
    title: "Test Issue",
    description: "This is a test description",
    steps: "1. Do something\n2. See error",
    version: "v1.0.0",
    system: "Linux/amd64",
    timezone: "MST -07:00",
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

v1.0.0

## System

Linux/amd64, MST -07:00`);
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

  describe("generateMailtoUrl", () => {
    it("generates plaintext mailto url without steps or diagnostics", () => {
      const url = generateMailtoUrl("support@example.com", mockIssueData);

      expect(url).toContain("mailto:support@example.com?subject=Test%20Issue&body=");
      const body = decodeURIComponent(url.split("&body=")[1]);
      expect(body).toBe(`This is a test description

Version: v1.0.0

System: Linux/amd64, MST -07:00`);
    });

    it("stays below the windows mailto limit at maximum input length", () => {
      // umlaut-dense german text, each umlaut costs 6 chars percent-encoded
      const text = "Fehlermeldung: Ladepunkt überprüfen, Zählerstände größer als üblich. ";
      const fill = (length: number) =>
        text.repeat(Math.ceil(length / text.length)).slice(0, length);

      const url = generateMailtoUrl("support@installer-example.com", {
        ...mockIssueData,
        title: fill(MAX_MAIL_TITLE_LENGTH),
        description: fill(MAX_MAIL_DESCRIPTION_LENGTH),
      });

      expect(url.length).toBeLessThan(2000);
    });
  });

  describe("generateDebugFile", () => {
    it("contains version, system and enabled sections", () => {
      const content = generateDebugFile(mockIssueData, mockSections);

      expect(content).toContain("# evcc debug information");
      expect(content).toContain("Version: v1.0.0");
      expect(content).toContain("System: Linux/amd64, MST -07:00");
      expect(content).toContain("## Configuration (YAML)");
      expect(content).toContain("## Logs");
      expect(content).not.toContain("## System State");
    });
  });
});
