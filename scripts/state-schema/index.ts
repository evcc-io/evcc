import { buildSchemas } from "./schemas";
import { writeStateSchemas, bundleMcpJson } from "./openapi";

const command = process.argv[2] ?? "generate";

switch (command) {
  case "generate": {
    const schemas = buildSchemas();
    await writeStateSchemas(schemas);
    await bundleMcpJson();
    console.log(
      `${Object.keys(schemas.defs).length} state schemas → openapi.state.yaml + mcp/openapi.json`
    );
    break;
  }
  case "dump": {
    console.log(JSON.stringify(buildSchemas().defs, null, 2));
    break;
  }
  case "validate": {
    const source = process.argv[3];
    if (!source) {
      console.error("usage: npm run openapi -- validate <payload.json | url>");
      process.exit(1);
    }
    const { validate } = await import("./validate");
    if (!(await validate(source))) process.exit(1);
    break;
  }
  default:
    console.error(`unknown command: ${command}`);
    process.exit(1);
}
