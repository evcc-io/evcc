import { describe, expect, test } from "vite-plus/test";
import { highlightAndSortChangelog } from "./highlightChangelog";

const html = `
<h1>v0.303.1</h1>
<h2>Changelog</h2>
<h3>Other Changes</h3>
<ul>
<li>Home Assistant: allow switch for enable/disable</li>
<li>Nexblue: remove broken 1p3p</li>
<li>Optimizer: return infeasable error</li>
</ul>
<h3>Bug Fixes</h3>
<ul>
<li>HomeAssistant: fix changelog</li>
<li>Optimizer: fix invalid battery capacity</li>
</ul>
`;

describe("highlightAndSortChangelog", () => {
  test("returns markup unchanged without active features", () => {
    expect(highlightAndSortChangelog(html, [])).toBe(html);
  });

  test("returns empty string unchanged", () => {
    expect(highlightAndSortChangelog("", ["OpenDTU"])).toBe("");
  });

  test("marks and moves matching entries to the top of each list", () => {
    const result = highlightAndSortChangelog(html, ["Home Assistant"]);

    const doc = new DOMParser().parseFromString(result, "text/html");
    const lists = doc.querySelectorAll("ul");

    // first list: "Home Assistant: ..." moved to the top and marked
    const firstItems = Array.from(lists[0].children);
    expect(firstItems[0].textContent).toContain("Home Assistant: allow switch");
    expect(firstItems[0].classList.contains("changelog-relevant")).toBe(true);
    expect(firstItems[1].classList.contains("changelog-relevant")).toBe(false);
    expect(firstItems[2].classList.contains("changelog-relevant")).toBe(false);

    // second list: "HomeAssistant: ..." (no space) still matches, moved to top
    const secondItems = Array.from(lists[1].children);
    expect(secondItems[0].textContent).toContain("HomeAssistant: fix changelog");
    expect(secondItems[0].classList.contains("changelog-relevant")).toBe(true);
  });

  test("preserves relative order when nothing matches", () => {
    const result = highlightAndSortChangelog(html, ["Sigenergy"]);
    const doc = new DOMParser().parseFromString(result, "text/html");
    const firstItems = Array.from(doc.querySelectorAll("ul")[0].children);
    expect(firstItems[0].textContent).toContain("Home Assistant: allow switch");
    expect(firstItems.some((li) => li.classList.contains("changelog-relevant"))).toBe(false);
  });

  test("matches type keys as well as product names", () => {
    const result = highlightAndSortChangelog(html, ["opendtu", "nexblue"]);
    const doc = new DOMParser().parseFromString(result, "text/html");
    const firstItems = Array.from(doc.querySelectorAll("ul")[0].children);
    expect(firstItems[0].textContent).toContain("Nexblue: remove broken");
    expect(firstItems[0].classList.contains("changelog-relevant")).toBe(true);
  });
});
