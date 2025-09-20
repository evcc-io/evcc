export interface IssueData {
  title: string;
  description: string;
  steps: string;
  version: string;
}

export interface SectionData {
  included: boolean;
  content: string;
}

export interface Sections {
  yamlConfig: SectionData;
  uiConfig: SectionData;
  state: SectionData;
  logs: SectionData;
}

export interface GitHubContent {
  body: string;
  additional?: string;
}

// First level items are joint with empty line (\n\n), second level with line wrap (\n)
export type Template = (string | string[])[];

export type HelpType = "discussion" | "issue";
