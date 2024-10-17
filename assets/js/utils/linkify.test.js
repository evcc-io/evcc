import linkify from "./linkify";
import { describe, expect, test } from "vitest";

describe("linkify", () => {
  test("should wrap links", () => {
    expect(linkify("https://example.com")).eq(
      `<a href="https://example.com" target="_blank">https://example.com</a>`
    );
  });
  test("with surrounding text", () => {
    expect(linkify("hello http://foo.bar/ world")).eq(
      `hello <a href="http://foo.bar/" target="_blank">http://foo.bar/</a> world`
    );
  });
  test("with query and hash", () => {
    expect(linkify("a http://b.c/?d=e#f g")).eq(
      `a <a href="http://b.c/?d=e#f" target="_blank">http://b.c/?d=e#f</a> g`
    );
  });
  test("with multiple links", () => {
    expect(linkify("hello http://foo.bar/ world https://bar.baz/ tadda!")).eq(
      `hello <a href="http://foo.bar/" target="_blank">http://foo.bar/</a> world <a href="https://bar.baz/" target="_blank">https://bar.baz/</a> tadda!`
    );
  });
});
