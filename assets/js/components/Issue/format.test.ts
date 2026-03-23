import { describe, it, expect } from "vitest";
import { formatJson } from "./format";

describe("formatJson", () => {
  it("formats basic object with sorted keys", () => {
    const obj = { foo: "bar", baz: 123 };
    const result = formatJson(obj);

    expect(result).toBe(`{
  "baz": 123,
  "foo": "bar"
}`);
  });

  it("sorts id and name first", () => {
    const obj = { zebra: 1, name: "test", id: 42, alpha: 2 };
    const result = formatJson(obj);

    expect(result).toBe(`{
  "id": 42,
  "name": "test",
  "alpha": 2,
  "zebra": 1
}`);
  });

  it("expands arrays with expand keys", () => {
    const obj = {
      other: ["baz", "qux"],
      items: ["foo", "bar"],
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
      other: { gamma: 3 },
      config: { beta: 2, alpha: 1 },
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

  it("keeps nested objects in arrays single-lined with sorted keys", () => {
    const obj = {
      items: [
        { foo: "bar", baz: 123 },
        { qux: "test", id: 1, num: 456 },
      ],
    };
    const result = formatJson(obj, ["items"]);

    expect(result).toBe(`{
  "items": [
    {"baz":123,"foo":"bar"},
    {"id":1,"num":456,"qux":"test"}
  ]
}`);
  });
});
