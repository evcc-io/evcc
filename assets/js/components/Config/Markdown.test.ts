import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vite-plus/test";
import Markdown from "./Markdown.vue";

const render = (markdown: string) => mount(Markdown, { props: { markdown } }).html();

describe("Markdown", () => {
  test("keeps underscores inside words as text", () => {
    const html = render("charge_before_export and a_b_c stay flat");
    expect(html).toContain("charge_before_export and a_b_c stay flat");
    expect(html).not.toContain("<em>");
  });

  test("still emphasises whole words", () => {
    expect(render("so _early_ and __bold__")).toContain(
      "so <em>early</em> and <strong>bold</strong>"
    );
  });

  test("keeps underscores inside code", () => {
    expect(render("`charge_before_export`")).toContain("<code>charge_before_export</code>");
  });
});
