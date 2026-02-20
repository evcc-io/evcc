/** Matches {placeholder} names (letters, numbers, underscores) */
const PLACEHOLDER_PATTERN = /\{(\w+)\}/g;

/** Extract {placeholder} names from template */
export function extractPlaceholders(template: string): string[] {
  const matches = template.match(PLACEHOLDER_PATTERN);
  if (!matches) return [];
  return matches.map((match) => match.slice(1, -1));
}

/** Replace {placeholders} with URL-encoded values */
export function replacePlaceholders(template: string, values: Record<string, string>): string {
  return template.replace(PLACEHOLDER_PATTERN, (match, key) => {
    const value = values[key];
    if (value === undefined) {
      return match;
    }
    return encodeURIComponent(value);
  });
}
