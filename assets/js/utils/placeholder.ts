/** Matches {placeholder} names (letters, numbers, underscores); a trailing ? marks it optional */
const PLACEHOLDER_PATTERN = /\{(\w+)(\??)\}/g;

/** Extract {placeholder} names from template */
export function extractPlaceholders(template: string): string[] {
  return [...template.matchAll(PLACEHOLDER_PATTERN)].map((match) => match[1]!);
}

/** Replace {placeholders} with URL-encoded values; missing optional {placeholders?} become empty */
export function replacePlaceholders(template: string, values: Record<string, string>): string {
  return template.replace(PLACEHOLDER_PATTERN, (match, key, optional) => {
    const value = values[key];
    if (value === undefined) {
      return optional ? "" : match;
    }
    return encodeURIComponent(value);
  });
}
