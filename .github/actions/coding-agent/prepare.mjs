import { randomUUID } from "node:crypto";
import { appendFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

export function prepare(env) {
	const model = (env.ACTIONS_MODEL || "").trim();
	const agent =
		!model || /^(claude-[\w.-]+|opus|sonnet|haiku|opusplan)(\[1m\])?$/.test(model)
			? "claude"
			: /^(gpt-[\w.-]+|azure\/[\w.-]+)$/.test(model)
				? "opencode"
				: null;
	if (!agent)
		throw new Error("ACTIONS_MODEL must be a Claude model, gpt-* deployment or azure/<deployment>");
	if (!["review", "label", "triage", "analyze", "fix"].includes(env.PROFILE)) {
		throw new Error("Unknown coding-agent profile");
	}
	if (
		!/^[1-9][0-9]*$/.test(env.NUMBER || "") ||
		!/^[\w.-]+\/[\w.-]+$/.test(env.REPOSITORY || "") ||
		(env.COMMENT_ID && !/^[1-9][0-9]*$/.test(env.COMMENT_ID))
	) {
		throw new Error("Invalid coding-agent target");
	}
	if (!env.AGENT_PROMPT?.trim()) throw new Error("Missing coding-agent prompt");
	if (!env.ACTION_PATH) throw new Error("Missing coding-agent action path");

	let endpoint = "";
	if (agent === "claude") {
		if (!env.CLAUDE_CODE_OAUTH_TOKEN)
			throw new Error("Configure the CLAUDE_CODE_OAUTH_TOKEN secret");
	} else {
		if (!env.AZURE_OPENAI_API_KEY) throw new Error("Configure the AZURE_OPENAI_API_KEY secret");
		let url;
		try {
			url = new URL(env.AZURE_OPENAI_ENDPOINT);
		} catch {
			/* validated below */
		}
		if (
			!url ||
			url.protocol !== "https:" ||
			!/^\/openai\/v1\/?$/.test(url.pathname) ||
			url.username ||
			url.password ||
			url.search ||
			url.hash
		) {
			throw new Error("Set AZURE_OPENAI_ENDPOINT to https://RESOURCE.openai.azure.com/openai/v1");
		}
		// The bundled Azure SDK appends /v1 for Azure OpenAI resource hosts.
		if (url.hostname.endsWith(".openai.azure.com")) url.pathname = "/openai";
		endpoint = url.toString().replace(/\/$/, "");
	}

	const write = ["triage", "fix"].includes(env.PROFILE);
	const commands = ["gh pr view *", "gh pr diff *"];
	if (env.PROFILE === "label") {
		commands.push("gh label list*", `gh pr edit ${env.NUMBER} --add-label *`);
	} else {
		commands.push("git log *");
		if (env.PROFILE === "review") {
			commands.push(`gh pr comment ${env.NUMBER} *`);
		} else {
			commands.push(
				"gh label list*",
				"gh issue view *",
				"git diff *",
				"git status*",
				`gh issue comment ${env.NUMBER} *`
			);
			if (env.COMMENT_ID) {
				commands.push(
					`gh api repos/${env.REPOSITORY}/issues/comments/${env.COMMENT_ID} --jq .body`
				);
			}
			if (env.PROFILE === "triage") commands.push(`gh issue edit ${env.NUMBER} *`);
		}
	}
	if (write) {
		commands.push(
			"gh pr create *",
			"git checkout *",
			"git switch *",
			"git add *",
			"git commit *",
			"git push *"
		);
	}
	const permission = {
		"*": "deny",
		read: "allow",
		glob: "allow",
		grep: "allow",
		edit: write ? "allow" : "deny",
		bash: { "*": "deny", ...Object.fromEntries(commands.map((command) => [command, "allow"])) },
	};
	const tools = ["Read", "Grep", "Glob", ...(write ? ["Edit", "Write"] : [])];
	const allowed = [...tools, ...commands.map((command) => `Bash(${command})`)];
	let args = `--tools "${[...tools, "Bash"].join(",")}" --allowed-tools "${allowed.join(",")}" --disallowed-tools "Task,Agent" --max-turns 100`;
	if (agent === "claude" && model) args += ` --model "${model}"`;
	const attribution =
		agent === "claude"
			? "🤖 Generated with [Claude Code](https://claude.com/claude-code)"
			: "🤖 Generated with [OpenCode](https://opencode.ai)";
	const hook = [process.execPath, join(env.ACTION_PATH, "guard.mjs")]
		.map((part) => `'${part.replaceAll("'", "'\\''")}'`)
		.join(" ");
	return {
		agent,
		model: model.replace(/^azure\//, ""),
		endpoint,
		permission: JSON.stringify(permission),
		claude_args: args,
		claude_settings: JSON.stringify({
			hooks: { PreToolUse: [{ matcher: "Bash", hooks: [{ type: "command", command: hook }] }] },
		}),
		prompt: `${env.AGENT_PROMPT.trim()}\n\nEnd every GitHub comment, review and PR body with this attribution footer:\n${attribution}`,
	};
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
	try {
		const values = prepare(process.env);
		const delimiter = randomUUID();
		appendFileSync(
			process.env.GITHUB_OUTPUT,
			Object.entries(values)
				.map(([key, value]) => `${key}<<${delimiter}\n${value}\n${delimiter}\n`)
				.join("")
		);
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}
