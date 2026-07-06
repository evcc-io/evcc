import { buildSchemas } from "./schemas";
import { spliceOpenapi } from "./openapi";

const command = process.argv[2] ?? "generate";

switch (command) {
  case "generate": {
    const schemas = buildSchemas();
    const changed = await spliceOpenapi(schemas);
    console.log(
      changed
        ? `server/openapi.yaml updated (${Object.keys(schemas.defs).length} schemas)`
        : "server/openapi.yaml unchanged"
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
