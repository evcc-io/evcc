import { readFileSync } from "node:fs";
import { parse } from "yaml";
import { Ajv2020 } from "ajv/dist/2020.js";
import addFormats from "ajv-formats";

const STATE_SCHEMAS_PATH = "server/openapi.state.yaml";

// intentionally undocumented experimental structures
const IGNORE = new Set(["$.evopt", "$.evopt-batteries"]);

type AnySchema = any;

function resolve(schema: AnySchema, schemas: Record<string, AnySchema>): AnySchema {
  if (typeof schema?.$ref === "string") {
    return resolve(schemas[schema.$ref.split("/").pop()!], schemas);
  }
  return schema ?? {};
}

// report payload keys that have no schema property, additionalProperties:true hides them from ajv
function coverage(
  schema: AnySchema,
  data: any,
  path: string,
  schemas: Record<string, AnySchema>,
  report: Set<string>
): void {
  const s = resolve(schema, schemas);
  if (s.anyOf) {
    if (data === null) return;
    const branch = s.anyOf.find((b: AnySchema) => resolve(b, schemas).type !== "null");
    if (branch) coverage(branch, data, path, schemas, report);
    return;
  }
  if (Array.isArray(data)) {
    if (s.items) data.forEach((item) => coverage(s.items, item, `${path}[]`, schemas, report));
    return;
  }
  if (data !== null && typeof data === "object") {
    if (!s.properties) return;
    for (const key of Object.keys(data)) {
      if (s.properties[key]) {
        coverage(s.properties[key], data[key], `${path}.${key}`, schemas, report);
      } else if (!IGNORE.has(`${path}.${key}`)) {
        report.add(`${path}.${key}`);
      }
    }
  }
}

// openapi uses `nullable: true`, json schema expects explicit null in the type
function expandNullable(node: any): void {
  if (Array.isArray(node)) {
    node.forEach(expandNullable);
  } else if (node && typeof node === "object") {
    if (node.nullable === true) {
      delete node.nullable;
      if (typeof node.type === "string") {
        node.type = [node.type, "null"];
      } else if (Array.isArray(node.allOf)) {
        node.anyOf = [...node.allOf, { type: "null" }];
        delete node.allOf;
      }
    }
    Object.values(node).forEach(expandNullable);
  }
}

// validate a /api/state payload against the State schema in server/openapi.state.yaml
export function validateState(payload: any): { errors: string[]; undocumented: string[] } {
  const doc = parse(readFileSync(STATE_SCHEMAS_PATH, "utf8"));
  const schemas: Record<string, AnySchema> = doc.components.schemas;
  expandNullable(schemas);

  const ajv = new Ajv2020({ strict: false, allErrors: true });
  addFormats(ajv);
  const validateFn = ajv.compile({
    $ref: "#/components/schemas/State",
    components: { schemas },
  });

  const errors = validateFn(payload)
    ? []
    : (validateFn.errors ?? []).map((err) => `${err.instancePath || "/"} ${err.message}`);

  const undocumented = new Set<string>();
  coverage(schemas["State"], payload, "$", schemas, undocumented);

  return { errors, undocumented: [...undocumented].sort() };
}

export async function validate(source: string): Promise<boolean> {
  let payload: any;
  if (source.startsWith("http")) {
    const res = await fetch(`${source.replace(/\/$/, "")}/api/state`);
    payload = await res.json();
  } else {
    payload = JSON.parse(readFileSync(source, "utf8"));
  }

  const { errors, undocumented } = validateState(payload);

  if (errors.length > 0) {
    console.error(`${source}: schema violations`);
    for (const err of errors.slice(0, 30)) console.error(`  ${err}`);
  }
  if (undocumented.length > 0) {
    console.log(`${source}: ${undocumented.length} undocumented keys`);
    for (const key of undocumented) console.log(`  ${key}`);
  }
  if (errors.length === 0 && undocumented.length === 0) console.log(`${source}: ok`);

  return errors.length === 0;
}
