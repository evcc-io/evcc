import { readFileSync, writeFileSync } from "node:fs";
import { stringify } from "yaml";
import prettier from "prettier";
import type { StateSchemas } from "./schemas";

const OPENAPI_PATH = "server/openapi.yaml";
const BEGIN = "# GENERATED_STATE_SCHEMAS_BEGIN";
const END = "# GENERATED_STATE_SCHEMAS_END";
const INDENT = "    "; // components.schemas entry level

function markerBounds(text: string): { begin: number; end: number } {
  if (text.split(BEGIN).length !== 2 || text.split(END).length !== 2) {
    throw new Error(`expected exactly one ${BEGIN} and one ${END} marker in ${OPENAPI_PATH}`);
  }
  const begin = text.indexOf("\n", text.indexOf(BEGIN)) + 1;
  const end = text.lastIndexOf("\n", text.indexOf(END)) + 1;
  if (begin > end) throw new Error("markers out of order");
  return { begin, end };
}

// schema names defined outside the generated block
function handWrittenNames(text: string, begin: number, end: number): Set<string> {
  const outside = text.slice(0, begin) + text.slice(end);
  const schemasSection = outside.match(/^ {2}schemas:\n([\s\S]*?)(?=^ {2}[a-zA-Z]|(?![\s\S]))/m);
  if (!schemasSection) throw new Error("components.schemas section not found");
  const names = new Set<string>();
  for (const line of schemasSection[1].matchAll(/^ {4}([a-zA-Z0-9._-]+):/gm)) {
    names.add(line[1]);
  }
  return names;
}

export async function spliceOpenapi(schemas: StateSchemas): Promise<boolean> {
  const text = readFileSync(OPENAPI_PATH, "utf8");
  const { begin, end } = markerBounds(text);

  const existing = handWrittenNames(text, begin, end);
  for (const name of Object.keys(schemas.defs)) {
    if (existing.has(name)) {
      throw new Error(`generated schema "${name}" collides with hand-written component`);
    }
  }

  const block = stringify(schemas.defs, { lineWidth: 0, aliasDuplicateObjects: false })
    .trimEnd()
    .split("\n")
    .map((line) => (line.length > 0 ? INDENT + line : line))
    .join("\n");

  const spliced = text.slice(0, begin) + block + "\n" + text.slice(end);

  const config = await prettier.resolveConfig(OPENAPI_PATH);
  const formatted = await prettier.format(spliced, { ...config, filepath: OPENAPI_PATH });

  if (formatted === text) return false;
  writeFileSync(OPENAPI_PATH, formatted);
  return true;
}
