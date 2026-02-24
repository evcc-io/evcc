/**
 * Custom JSON formatter that keeps arrays of primitive values on a single line
 */
export function formatCompactJson(obj: any, indent: number = 2): string {
  const spaces = " ".repeat(indent);

  function stringify(value: any, depth: number): string {
    const currentIndent = spaces.repeat(depth);
    const nextIndent = spaces.repeat(depth + 1);

    if (value === null) return "null";
    if (value === undefined) return "undefined";

    // Handle primitive types
    if (typeof value !== "object") {
      return JSON.stringify(value);
    }

    // Handle arrays
    if (Array.isArray(value)) {
      // Check if all elements are primitives (numbers, strings, booleans, null)
      const allPrimitive = value.every(
        (item) => item === null || item === undefined || typeof item !== "object"
      );

      if (allPrimitive && value.length > 0) {
        // Format primitive array on single line
        const items = value.map((item) => JSON.stringify(item)).join(", ");
        return `[${items}]`;
      } else if (value.length === 0) {
        return "[]";
      } else {
        // Format complex array with indentation
        const items = value.map((item) => `${nextIndent}${stringify(item, depth + 1)}`).join(",\n");
        return `[\n${items}\n${currentIndent}]`;
      }
    }

    // Handle objects
    const entries = Object.entries(value);
    if (entries.length === 0) {
      return "{}";
    }

    const items = entries
      .map(([key, val]) => {
        const formattedValue = stringify(val, depth + 1);
        return `${nextIndent}${JSON.stringify(key)}: ${formattedValue}`;
      })
      .join(",\n");

    return `{\n${items}\n${currentIndent}}`;
  }

  return stringify(obj, 0);
}
