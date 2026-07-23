import { describe, expect, test } from "vitest";
import { stringifyQuery } from "./router";

describe("stringifyQuery", () => {
  test("empty inputs", () => {
    expect(stringifyQuery()).toBe("");
    expect(stringifyQuery({})).toBe("");
    expect(stringifyQuery({ items: [] })).toBe("");
  });

  test("basic key-value pairs", () => {
    expect(stringifyQuery({ foo: "bar" })).toBe("foo=bar");
    expect(stringifyQuery({ foo: "bar", baz: "qux" })).toBe("foo=bar&baz=qux");
  });

  test("URI encoding", () => {
    expect(stringifyQuery({ key: "value with spaces" })).toBe("key=value%20with%20spaces");
    expect(stringifyQuery({ key: "special&chars=here" })).toBe("key=special%26chars%3Dhere");
    expect(stringifyQuery({ "key[0]": "value" })).toBe("key[0]=value");
  });

  test("array values", () => {
    expect(stringifyQuery({ items: ["a", "b", "c"] })).toBe("items=a&items=b&items=c");
    expect(stringifyQuery({ items: ["foo bar", "baz qux"] })).toBe(
      "items=foo%20bar&items=baz%20qux"
    );
    expect(stringifyQuery({ items: ["a", "", "c"] })).toBe("items=a&items&items=c");
  });

  test("falsy values", () => {
    expect(stringifyQuery({ key: "" })).toBe("key");
    expect(stringifyQuery({ key: null })).toBe("key");
    expect(stringifyQuery({ key: undefined })).toBe("key");
    expect(stringifyQuery({ key: 0 })).toBe("key");
    expect(stringifyQuery({ key: false })).toBe("key");
  });

  test("truthy values", () => {
    expect(stringifyQuery({ key: "value" })).toBe("key=value");
    expect(stringifyQuery({ key: 123 })).toBe("key=123");
    expect(stringifyQuery({ key: true })).toBe("key=true");
  });

  test("complex scenarios", () => {
    expect(stringifyQuery({ a: "value", b: "", c: "another" })).toBe("a=value&b&c=another");
    expect(stringifyQuery({ filter: ["tag1", "tag2"], sort: "name", page: 1, empty: "" })).toBe(
      "filter=tag1&filter=tag2&sort=name&page=1&empty"
    );
    expect(stringifyQuery({ z: "last", a: "first", m: "middle" })).toBe("z=last&a=first&m=middle");
  });
});
