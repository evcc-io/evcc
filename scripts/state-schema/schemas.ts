import { createGenerator } from "ts-json-schema-generator";
import type { Schema } from "ts-json-schema-generator";

export interface StateSchemas {
  rootName: string;
  // schema per component name, root first, rest alphabetical
  defs: Record<string, Schema>;
}

// enums use SCREAMING_SNAKE in the frontend, schema names follow openapi PascalCase convention
const RENAME: Record<string, string> = {
  CHARGE_MODE: "ChargeMode",
  BATTERY_MODE: "BatteryMode",
  CURRENCY: "Currency",
  CHARGER_STATUS_REASON: "ChargerStatusReason",
  PHASE_ACTION: "PhaseAction",
  PV_ACTION: "PvAction",
  SMART_COST_TYPE: "SmartCostType",
  OCPP_STATION_STATUS: "OcppConnectionStatus",
  MODBUS_BAUDRATE: "ModbusBaudrate",
  MODBUS_COMSET: "ModbusComset",
  MODBUS_PROXY_READONLY: "ModbusProxyReadonly",
};

const VALID_NAME = /^[a-zA-Z0-9._-]+$/;
const ROOT = "State";

function buildRawSchema(): Schema {
  const generator = createGenerator({
    path: "assets/js/types/evcc.ts",
    tsconfig: "tsconfig.json",
    type: ROOT,
    jsDoc: "extended",
    extraTags: ["internal"],
    additionalProperties: true,
    topRef: false,
    skipTypeCheck: true,
    sortProps: false,
  });
  return generator.createSchema(ROOT);
}

type AnySchema = any;

function walk(node: any, visit: (schema: AnySchema) => void): void {
  if (Array.isArray(node)) {
    node.forEach((child) => walk(child, visit));
  } else if (node && typeof node === "object") {
    visit(node);
    Object.values(node).forEach((child) => walk(child, visit));
  }
}

// remove properties tagged @internal, they are UI-only and not part of the API payload
function stripInternal(schema: AnySchema): void {
  walk(schema, (node) => {
    if (!node.properties) return;
    for (const [key, prop] of Object.entries<AnySchema>(node.properties)) {
      if (prop?.internal === true) {
        delete node.properties[key];
        if (Array.isArray(node.required)) {
          node.required = node.required.filter((r: string) => r !== key);
          if (node.required.length === 0) delete node.required;
        }
      }
    }
  });
}

function collectRefs(schema: AnySchema): Set<string> {
  const refs = new Set<string>();
  walk(schema, (node) => {
    if (typeof node.$ref === "string") {
      refs.add(decodeURIComponent(node.$ref.replace("#/definitions/", "")));
    }
  });
  return refs;
}

// drop definitions that became unreachable after stripping @internal properties
function reachableDefs(
  root: AnySchema,
  defs: Record<string, AnySchema>
): Record<string, AnySchema> {
  const keep: Record<string, AnySchema> = {};
  const queue = [...collectRefs(root)];
  while (queue.length > 0) {
    const name = queue.shift()!;
    if (keep[name] || !defs[name]) continue;
    keep[name] = defs[name];
    queue.push(...collectRefs(defs[name]));
  }
  return keep;
}

// convert json-schema null unions to `nullable: true`, the 3.0 style used across openapi.yaml
// (kin-openapi, which validates the spec in CI, does not support 3.1 type arrays)
function normalizeNullables(schema: AnySchema): void {
  walk(schema, (node) => {
    if (Array.isArray(node.type) && node.type.includes("null")) {
      const rest = node.type.filter((t: string) => t !== "null");
      if (rest.length !== 1) throw new Error(`unsupported type union ${node.type}`);
      node.type = rest[0];
      node.nullable = true;
    }
    if (Array.isArray(node.anyOf) && node.anyOf.some((b: AnySchema) => b.type === "null")) {
      const rest = node.anyOf.filter((b: AnySchema) => b.type !== "null");
      delete node.anyOf;
      node.nullable = true;
      if (rest.length === 1 && !rest[0].$ref) {
        Object.assign(node, rest[0]);
      } else {
        node.allOf = rest;
      }
    }
    // openapi 3.0 has no `const`, use a single-value enum
    if ("const" in node) {
      node.enum = [node.const];
      delete node.const;
    }
    // openapi 3.0 uses a singular example
    if (Array.isArray(node.examples)) {
      node.example = node.examples[0];
      delete node.examples;
    }
  });
}

// point $refs at the final openapi component names
function rewriteRefs(schema: AnySchema, finalNames: Map<string, string>): void {
  walk(schema, (node) => {
    if (typeof node.$ref !== "string" || node.$ref.startsWith("#/components/")) return;
    const name = decodeURIComponent(node.$ref.replace("#/definitions/", ""));
    const renamed = finalNames.get(name);
    if (!renamed) throw new Error(`unresolved $ref "${node.$ref}"`);
    node.$ref = `#/components/schemas/${renamed}`;
  });
}

export function buildSchemas(): StateSchemas {
  const raw = buildRawSchema() as AnySchema;
  const { definitions = {}, $schema, ...root } = raw;

  stripInternal(root);
  Object.values(definitions as Record<string, AnySchema>).forEach(stripInternal);

  const defs = reachableDefs(root, definitions);

  const finalNames = new Map<string, string>();
  for (const name of Object.keys(defs)) {
    const renamed = RENAME[name] ?? name;
    if (!VALID_NAME.test(renamed)) {
      throw new Error(`invalid component name "${renamed}", rename the type in evcc.ts`);
    }
    if ([...finalNames.values()].includes(renamed)) {
      throw new Error(`duplicate component name "${renamed}" after renaming`);
    }
    finalNames.set(name, renamed);
  }

  rewriteRefs(root, finalNames);
  Object.values(defs).forEach((schema) => rewriteRefs(schema, finalNames));

  normalizeNullables(root);
  Object.values(defs).forEach(normalizeNullables);

  const result: Record<string, Schema> = { [ROOT]: root };
  for (const [name, schema] of Object.entries(defs)
    .map(([name, schema]) => [finalNames.get(name)!, schema] as const)
    .sort(([a], [b]) => a.localeCompare(b))) {
    result[name] = schema;
  }

  const serialized = JSON.stringify(result);
  if (serialized.includes("#/definitions/")) {
    throw new Error("unrewritten $ref to #/definitions/ left in output");
  }

  const ordered = Object.fromEntries(
    Object.entries(result).map(([name, schema]) => [name, orderKeys(schema)])
  );

  return { rootName: ROOT, defs: ordered };
}

const KEY_ORDER = [
  "$ref",
  "description",
  "type",
  "nullable",
  "enum",
  "format",
  "example",
  "items",
  "properties",
  "required",
  "additionalProperties",
  "anyOf",
  "allOf",
];

// stable key order for readable yaml diffs, property order itself is preserved
function orderKeys(node: any): any {
  if (Array.isArray(node)) return node.map(orderKeys);
  if (!node || typeof node !== "object") return node;
  const keys = Object.keys(node).sort((a, b) => {
    const ia = KEY_ORDER.indexOf(a);
    const ib = KEY_ORDER.indexOf(b);
    return (ia === -1 ? KEY_ORDER.length : ia) - (ib === -1 ? KEY_ORDER.length : ib);
  });
  return Object.fromEntries(
    keys.map((key) => [
      key,
      key === "properties" ? mapValues(node[key], orderKeys) : orderKeys(node[key]),
    ])
  );
}

function mapValues(obj: Record<string, any>, fn: (v: any) => any): Record<string, any> {
  return Object.fromEntries(Object.entries(obj).map(([k, v]) => [k, fn(v)]));
}
