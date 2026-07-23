import { readFileSync, writeFileSync } from "node:fs";
import { parse, stringify } from "yaml";
import prettier from "prettier";
import { bundle, createConfig } from "@redocly/openapi-core";
import type { StateSchemas } from "./schemas";

const OPENAPI_PATH = "server/openapi.yaml";
const STATE_PATH = "server/openapi.state.yaml";
const MCP_JSON_PATH = "server/mcp/openapi.json";

const HEADER = `# GENERATED FILE - DO NOT EDIT (source: assets/js/types/evcc.ts, update: npm run openapi)
`;

export async function writeStateSchemas(schemas: StateSchemas): Promise<void> {
  const root = parse(readFileSync(OPENAPI_PATH, "utf8"));
  const handWritten = new Set(Object.keys(root.components?.schemas ?? {}));
  for (const name of Object.keys(schemas.defs)) {
    if (handWritten.has(name)) {
      throw new Error(`generated schema "${name}" collides with hand-written component`);
    }
  }

  const doc =
    HEADER +
    stringify(
      { components: { schemas: schemas.defs } },
      { lineWidth: 0, aliasDuplicateObjects: false }
    );

  const config = await prettier.resolveConfig(STATE_PATH);
  writeFileSync(STATE_PATH, await prettier.format(doc, { ...config, filepath: STATE_PATH }));
}

// inline the multi-file spec into the single json embedded by the MCP server
export async function bundleMcpJson(): Promise<void> {
  const { bundle: result } = await bundle({ ref: OPENAPI_PATH, config: await createConfig({}) });
  const doc = result.parsed as { servers?: unknown };
  delete doc.servers; // mcp server sets its own url
  writeFileSync(MCP_JSON_PATH, JSON.stringify(doc, null, 2) + "\n");
}
