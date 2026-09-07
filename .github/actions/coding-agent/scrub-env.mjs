import { normalizeCommand } from "./guard.mjs";

export default async () => ({
	"tool.execute.before": async (input, output) => {
		if (input.tool !== "bash") return;
		if (output.args?.run_in_background || output.args?.background) {
			throw new Error("Background commands are not allowed");
		}
		const command = normalizeCommand(output.args?.command);
		output.args.command = command;
	},
	"shell.env": async (_input, output) => {
		// The provider needs the inference key; agent shell commands do not.
		output.env.AZURE_OPENAI_API_KEY = "";
		output.env.AGENT_PROMPT = "";
		output.env.GH_PAGER = "cat";
		output.env.PAGER = "cat";
		output.env.GIT_PAGER = "cat";
		output.env.GH_FORCE_TTY = "";
	},
});
