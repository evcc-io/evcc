import { describe, expect, test } from "vite-plus/test";
import { cleanReasoning } from "./reasoning";

describe("cleanReasoning", () => {
  test("keeps plain reasoning", () => {
    expect(cleanReasoning("  I need the charging mode.  ")).toBe("I need the charging mode.");
  });

  test("drops a tool call and its payload", () => {
    const text = 'I will look it up.<|tool_call|>{"name":"getState","arguments":{}}';
    expect(cleanReasoning(text)).toBe("I will look it up.");
  });

  test("resumes after the call", () => {
    const text = 'Looking.<|tool_call|>{"name":"getState"}<|message|>The mode is pv.';
    expect(cleanReasoning(text)).toBe("Looking.The mode is pv.");
  });

  test("removes markers that are not calls", () => {
    expect(cleanReasoning("<|channel|>analysis<|message|>done")).toBe("analysisdone");
  });

  test("collapses the gaps left behind", () => {
    expect(cleanReasoning("a\n\n\n\nb")).toBe("a\n\nb");
  });
});
