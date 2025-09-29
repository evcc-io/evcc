import { describe, it, expect } from "vitest";
import { formatJson } from "./format";

describe("formatJson", () => {
  it("formats basic object", () => {
    const obj = { foo: "bar", baz: 123 };
    const result = formatJson(obj);

    expect(result).toBe(`{
  "foo": "bar",
  "baz": 123
}`);
  });

  it("expands arrays with expand keys", () => {
    const obj = {
      items: ["foo", "bar"],
      other: ["baz", "qux"],
    };
    const result = formatJson(obj, ["items"]);

    expect(result).toBe(`{
  "items": [
    "foo",
    "bar"
  ],
  "other": ["baz","qux"]
}`);
  });

  it("expands objects with expand keys", () => {
    const obj = {
      config: { alpha: 1, beta: 2 },
      other: { gamma: 3 },
    };
    const result = formatJson(obj, ["config"]);

    expect(result).toBe(`{
  "config": {
    "alpha": 1,
    "beta": 2
  },
  "other": {"gamma":3}
}`);
  });

  it("keeps nested objects in arrays single-lined", () => {
    const obj = {
      items: [
        { foo: "bar", baz: 123 },
        { qux: "test", num: 456 },
      ],
    };
    const result = formatJson(obj, ["items"]);

    expect(result).toBe(`{
  "items": [
    {"foo":"bar","baz":123},
    {"qux":"test","num":456}
  ]
}`);
  });
});
