export function formatJson(obj: any, expandKeys: string[] = []): string {
  if (!obj || typeof obj !== "object") {
    return JSON.stringify(obj, null, 2);
  }

  const lines: string[] = [];

  for (const [key, value] of Object.entries(obj)) {
    let valueStr: string;

    // Check if this key should be expanded (only if not empty)
    if (
      expandKeys.includes(key) &&
      (Array.isArray(value) || (typeof value === "object" && value !== null))
    ) {
      if (Array.isArray(value)) {
        if (value.length === 0) {
          // Keep empty arrays compact
          valueStr = "[]";
        } else {
          const arrayItems = value.map((item) => {
            const itemStr = JSON.stringify(item);
            return `    ${itemStr.replace(/\\n/g, "\n")}`;
          });
          valueStr = `[\n${arrayItems.join(",\n")}\n  ]`;
        }
      } else {
        // Object expansion
        const objEntries = Object.entries(value);
        if (objEntries.length === 0) {
          // Keep empty objects compact
          valueStr = "{}";
        } else {
          const objItems = objEntries.map(([k, v]) => {
            const itemStr = JSON.stringify(v);
            return `    ${JSON.stringify(k)}: ${itemStr.replace(/\\n/g, "\n")}`;
          });
          valueStr = `{\n${objItems.join(",\n")}\n  }`;
        }
      }
    } else {
      // Single line for everything else
      valueStr = JSON.stringify(value).replace(/\\n/g, "\n");
    }

    lines.push(`  ${JSON.stringify(key)}: ${valueStr}`);
  }

  return `{\n${lines.join(",\n")}\n}`;
}
