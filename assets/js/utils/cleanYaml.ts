// accepts a multiline yaml string. if it starts with a key, the key will be removed, and the indent level will be reduced by one
// example 1: "key: value" -> "value"
// example 2:
// """
// key:
//   foo: bar
// """
// will be transformed into:
// """
// foo: bar
// """
export function cleanYaml(text: string, removeKey: string) {
  if (!removeKey) return text;
  const result: string[] = [];

  const prefix = removeKey + ":";
  const lines = text.split("\n");

  // remove first comment lines
  while (lines[0].startsWith("#")) lines.shift();

  const [firstLine, ...restLines] = lines;

  if (!firstLine.startsWith(prefix)) {
    // does not start with key, skip
    return text;
  } else {
    const first = firstLine.slice(prefix.length).trim();
    if (first) {
      result.push(first);
    }
  }

  if (restLines.length > 0) {
    const indentChars = restLines[0].match(/^(\s+)/)?.[0] || "";
    restLines
      .map((l) => (l.startsWith(indentChars) ? l.slice(indentChars.length) : l))
      .map((l) => l.trimEnd())
      .forEach((l) => result.push(l));
  }

  return result.join("\n");
}
