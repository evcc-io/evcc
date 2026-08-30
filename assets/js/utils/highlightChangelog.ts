import { normalizeFeatureName } from "./activeFeatures";

const RELEVANT_CLASS = "changelog-relevant";

// evcc changelog entries conventionally start with the integration they affect,
// e.g. "Home Assistant: allow switch for enable/disable". Extract that prefix.
function entryPrefix(li: Element): string | null {
  const text = li.textContent || "";
  const idx = text.indexOf(":");
  if (idx === -1) return null;
  return normalizeFeatureName(text.slice(0, idx));
}

// highlights and reorders changelog list items so entries matching the user's
// configured features (chargers, meters, vehicles, tariffs, ...) are marked and
// moved to the top of their list. Leaves the markup untouched when no active
// features are known, e.g. an unauthenticated session.
export function highlightAndSortChangelog(html: string, activeFeatures: string[]): string {
  if (!html || !activeFeatures.length) return html;

  const activeSet = new Set(activeFeatures.map(normalizeFeatureName));

  const doc = new DOMParser().parseFromString(`<div>${html}</div>`, "text/html");
  const container = doc.body.firstElementChild;
  if (!container) return html;

  container.querySelectorAll("ul").forEach((ul) => {
    const items = Array.from(ul.children).map((li, index) => {
      const prefix = entryPrefix(li);
      const relevant = prefix !== null && activeSet.has(prefix);
      if (relevant) {
        li.classList.add(RELEVANT_CLASS);
      }
      return { li, relevant, index };
    });

    // nothing to reorder if no entry matched
    if (!items.some((item) => item.relevant)) return;

    items
      .slice()
      .sort((a, b) => {
        if (a.relevant !== b.relevant) return a.relevant ? -1 : 1;
        return a.index - b.index;
      })
      .forEach((item) => ul.appendChild(item.li));
  });

  return container.innerHTML;
}
