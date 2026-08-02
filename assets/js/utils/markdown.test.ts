import { describe, expect, test } from "vite-plus/test";
import { renderMarkdown } from "./markdown";

describe("renderMarkdown", () => {
  test("renders a table", () => {
    const html = renderMarkdown("| mode | power |\n| --- | ---: |\n| pv | 3.7 kW |");

    expect(html).toBe(
      "<table><thead><tr><th>mode</th><th>power</th></tr></thead>" +
        "<tbody><tr><td>pv</td><td>3.7 kW</td></tr></tbody></table>"
    );
  });

  test("renders inline markdown inside cells", () => {
    const html = renderMarkdown("| a |\n| --- |\n| **pv** |");
    expect(html).toContain("<td><strong>pv</strong></td>");
  });

  test("keeps surrounding markdown", () => {
    const html = renderMarkdown("# Modes\n\n| a |\n| --- |\n| pv |\n\ndone");

    expect(html).toContain("<h1>Modes</h1>");
    expect(html).toContain("<table>");
    expect(html).toContain("done");
  });

  test("pads and truncates ragged rows", () => {
    const html = renderMarkdown("| a | b |\n| --- | --- |\n| 1 |\n| 1 | 2 | 3 |");

    expect(html).toContain("<tr><td>1</td><td></td></tr>");
    expect(html).toContain("<tr><td>1</td><td>2</td></tr>");
  });

  test("leaves pipes in fenced code alone", () => {
    const html = renderMarkdown("```\n| a | b |\n| --- | --- |\n```");

    expect(html).not.toContain("<table>");
    expect(html).toContain("<code>");
  });

  test("leaves prose containing a pipe alone", () => {
    const html = renderMarkdown("use a | b to filter");
    expect(html).not.toContain("<table>");
  });

  test("opens links in a new window", () => {
    const html = renderMarkdown("[docs](https://docs.evcc.io)");
    expect(html).toContain('target="_blank" rel="noopener noreferrer"');
  });
});
