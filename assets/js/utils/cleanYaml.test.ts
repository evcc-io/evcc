import { describe, expect, test } from "vitest";
import { cleanYaml } from "./cleanYaml";

describe("cleanYaml", () => {
  test("removes key from single line", () => {
    const result = cleanYaml("key: value", "key");
    expect(result).toBe("value");
  });
  test("removes key from multi-line yaml", () => {
    const input = `key:\n  nested: value\n  another: thing`;
    const expected = `nested: value\nanother: thing`;
    const result = cleanYaml(input, "key");
    expect(result).toBe(expected);
  });
  test("keep untouched if key not found", () => {
    const result = cleanYaml("key: value", "not-found");
    expect(result).toBe("key: value");
  });
  test("trim whitespace at the end of the line", () => {
    const result = cleanYaml("key: \n  - foo   \n  - bar  ", "key");
    expect(result).toBe("- foo\n- bar");
  });
  test("should remove leading comment lines", () => {
    const result = cleanYaml("# this is\n# a comment\nkey: value", "key");
    expect(result).toBe("value");
  });
  test("should note remove leading comment lines if key is not found", () => {
    const result = cleanYaml("# this is\n# a comment\nnot-found: value", "key");
    expect(result).toBe("# this is\n# a comment\nnot-found: value");
  });
});
