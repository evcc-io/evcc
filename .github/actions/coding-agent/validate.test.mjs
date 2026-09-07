import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { test } from "node:test";
import YAML from "yaml";
import { prepare } from "./prepare.mjs";
import scrubEnv from "./scrub-env.mjs";

const readYaml = (path) => YAML.parse(readFileSync(new URL(path, import.meta.url), "utf8"));
const action = readYaml("./action.yml");
const env = {
	ACTION_PATH: "/trusted/action",
	PROFILE: "analyze",
	NUMBER: "123",
	REPOSITORY: "evcc-io/evcc",
	COMMENT_ID: "456",
	AGENT_PROMPT: "Analyze this issue.",
	CLAUDE_CODE_OAUTH_TOKEN: "claude-test-token",
	AZURE_OPENAI_API_KEY: "azure-test-key",
	AZURE_OPENAI_ENDPOINT: "https://example.openai.azure.com/openai/v1",
};

test("shell steps parse and runners are mutually exclusive", () => {
	for (const step of action.runs.steps.filter((step) => step.run)) {
		const result = spawnSync("bash", ["-n"], { input: step.run, encoding: "utf8" });
		assert.equal(result.status, 0, result.stderr);
	}
	assert.equal(
		action.runs.steps.find((step) => step.uses?.startsWith("anthropics/claude-code-action@"))?.if,
		"steps.prepare.outputs.agent == 'claude'"
	);
	assert.equal(action.runs.steps.at(-1).if, "steps.prepare.outputs.agent == 'opencode'");
	assert.equal(action.runs.steps[0].env.ACTIONS_MODEL, "${{ vars.ACTIONS_MODEL }}");
});

test("ACTIONS_MODEL selects the runner without workflow changes", () => {
	for (const [model, agent, deployment] of [
		["", "claude", ""],
		["opus", "claude", "opus"],
		["opusplan", "claude", "opusplan"],
		["sonnet[1m]", "claude", "sonnet[1m]"],
		["claude-opus-4-6", "claude", "claude-opus-4-6"],
		["claude-opus-4-6[1m]", "claude", "claude-opus-4-6[1m]"],
		["gpt-6-astra", "opencode", "gpt-6-astra"],
		["azure/custom-deployment", "opencode", "custom-deployment"],
	]) {
		const result = prepare({ ...env, ACTIONS_MODEL: model });
		assert.equal(result.agent, agent);
		assert.equal(result.model, deployment);
		assert.match(
			result.prompt,
			agent === "claude" ? /Generated with \[Claude Code\]/ : /Generated with \[OpenCode\]/
		);
		assert.equal(result.claude_args.includes("--model"), agent === "claude" && model !== "");
	}
	assert.doesNotThrow(() =>
		prepare({ ...env, AZURE_OPENAI_API_KEY: "", AZURE_OPENAI_ENDPOINT: "" })
	);
	assert.doesNotThrow(() =>
		prepare({ ...env, ACTIONS_MODEL: "gpt-6-astra", CLAUDE_CODE_OAUTH_TOKEN: "" })
	);
});

test("actor policy is independent of the selected runner", async () => {
	const step = action.runs.steps.find((step) => step.name === "Validate triggering actor");
	assert.equal(step.if, undefined);
	const AsyncFunction = Object.getPrototypeOf(async () => {}).constructor;
	const run = new AsyncFunction("github", "context", "core", "process", step.with.script);
	for (const profile of ["review", "label", "triage", "analyze", "fix"]) {
		for (const [actor, type] of [
			["contributor", "User"],
			["dependabot[bot]", "Bot"],
			["other[bot]", "Bot"],
			["Copilot", null],
		]) {
			let failed = false;
			await run(
				{
					rest: {
						users: {
							getByUsername: async () => {
								if (type === null) throw { status: 404 };
								return { data: { type } };
							},
						},
					},
				},
				{ actor },
				{
					setFailed: () => {
						failed = true;
					},
				},
				{ env: { PROFILE: profile } }
			);
			assert.equal(
				failed,
				type !== "User" && !(profile === "review" && actor === "dependabot[bot]")
			);
		}
	}
	await assert.rejects(
		run(
			{
				rest: {
					users: {
						getByUsername: async () => {
							throw new Error("API unavailable");
						},
					},
				},
			},
			{ actor: "contributor" },
			{},
			{ env: { PROFILE: "review" } }
		),
		/API unavailable/
	);
});

test("both runners install the command guard before execution", () => {
	const install = action.runs.steps.findIndex((step) => step.name === "Install command guard");
	const claude = action.runs.steps.findIndex((step) => step.name === "Run Claude Code");
	const opencode = action.runs.steps.findIndex((step) => step.name === "Run OpenCode");
	assert.ok(install > 0 && install < claude && install < opencode);
	assert.equal(action.runs.steps[install].if, undefined);
	assert.equal(
		action.runs.steps[claude].with.settings,
		"${{ steps.prepare.outputs.claude_settings }}"
	);
	const settings = JSON.parse(prepare(env).claude_settings);
	assert.equal(settings.hooks.PreToolUse[0].matcher, "Bash");
	assert.match(settings.hooks.PreToolUse[0].hooks[0].command, /\/trusted\/action\/guard\.mjs/);
});

test("invalid configuration fails closed", () => {
	for (const changes of [
		{ ACTIONS_MODEL: "unknown" },
		{ ACTIONS_MODEL: 'claude-opus-4-6" --unsafe' },
		{ PROFILE: "unknown" },
		{ NUMBER: "" },
		{ NUMBER: "123;echo bad" },
		{ REPOSITORY: "bad repo" },
		{ COMMENT_ID: "invalid" },
		{ AGENT_PROMPT: "" },
		{ ACTION_PATH: "" },
		{ CLAUDE_CODE_OAUTH_TOKEN: "" },
		{ ACTIONS_MODEL: "gpt-6-astra", AZURE_OPENAI_API_KEY: "" },
	]) {
		assert.throws(() => prepare({ ...env, ...changes }));
	}
	for (const endpoint of [
		"",
		"http://example/openai/v1",
		"https://example.openai.azure.com",
		"https://user:pass@example/openai/v1",
		"https://example/openai/v1?x=1",
	]) {
		assert.throws(() =>
			prepare({ ...env, ACTIONS_MODEL: "gpt-6-astra", AZURE_OPENAI_ENDPOINT: endpoint })
		);
	}
});

test("Azure endpoint normalization matches the bundled SDK", () => {
	for (const [endpoint, expected] of [
		["https://example.openai.azure.com/openai/v1", "https://example.openai.azure.com/openai"],
		["https://example.openai.azure.com/openai/v1/", "https://example.openai.azure.com/openai"],
		[
			"https://example.services.ai.azure.com/openai/v1",
			"https://example.services.ai.azure.com/openai/v1",
		],
	]) {
		assert.equal(
			prepare({ ...env, ACTIONS_MODEL: "gpt-6-astra", AZURE_OPENAI_ENDPOINT: endpoint }).endpoint,
			expected
		);
	}
});

test("both runners enforce the same profile command boundaries", () => {
	for (const profile of ["review", "label", "triage", "analyze", "fix"]) {
		const result = prepare({ ...env, PROFILE: profile });
		const permission = JSON.parse(result.permission);
		const write = ["triage", "fix"].includes(profile);
		assert.equal(permission["*"], "deny");
		assert.equal(permission.bash["*"], "deny");
		assert.equal(permission.edit, write ? "allow" : "deny");
		assert.equal(permission.bash["git push *"] === "allow", write);
		assert.equal(permission.bash["gh issue edit 123 *"] === "allow", profile === "triage");
		assert.equal(permission.bash["gh pr edit 123 --add-label *"] === "allow", profile === "label");
		assert.equal(permission.bash["gh pr comment 123 *"] === "allow", profile === "review");
		assert.equal(
			permission.bash["gh issue comment 123 *"] === "allow",
			["triage", "analyze", "fix"].includes(profile)
		);
		assert.equal(permission.bash["gh api *"], undefined);
		assert.match(result.claude_args, /--disallowed-tools "Task,Agent"/);
		for (const command of Object.keys(permission.bash).filter((command) => command !== "*")) {
			assert.ok(result.claude_args.includes(`Bash(${command})`));
		}
	}
});

test("agent shells do not inherit the inference key or prompt", async () => {
	const hooks = await scrubEnv();
	const output = { env: { GH_TOKEN: "scoped-token" } };
	await hooks["shell.env"]({}, output);
	assert.equal(output.env.AZURE_OPENAI_API_KEY, "");
	assert.equal(output.env.AGENT_PROMPT, "");
	assert.equal(output.env.GH_TOKEN, "scoped-token");
});

test("API errors and unfinished turns fail the OpenCode action", () => {
	const filter = action.runs.steps.at(-1).run.match(/jq -s -e '([^']+)'/)[1];
	for (const [events, expected] of [
		[[{ type: "step_finish", part: { reason: "stop" } }], 0],
		[[], 1],
		[[{ type: "step_finish", part: { reason: "length" } }], 1],
		[[{ type: "step_finish", part: { reason: "stop" } }, { type: "error" }], 1],
	]) {
		assert.equal(
			spawnSync("jq", ["-s", "-e", filter], { input: events.map(JSON.stringify).join("\n") })
				.status,
			expected
		);
	}
});

test("workflow identities and task interfaces are stable", () => {
	const review = readYaml("../../workflows/claude-code-review-run.yml");
	assert.deepEqual(review.on.workflow_run.workflows, ["Claude Code Review"]);
	assert.match(
		review.jobs["claude-review"].steps.at(-1).with.script,
		/context: 'Claude Code Review'/
	);
	for (const [name, job] of [
		["claude-code-review-run.yml", "claude-review"],
		["claude-issue-agent-run.yml", "run"],
		["claude-issue-agent.yml", "pr-label"],
	]) {
		const workflow = readYaml(`../../workflows/${name}`);
		const step = workflow.jobs[job].steps.find(
			(step) => step.uses === "./.github/actions/coding-agent"
		);
		assert.ok(step.with.profile);
		assert.ok(step.with.number);
		assert.equal(step.with.permission, undefined);
		assert.equal(step.with.claude_args, undefined);
	}
	for (const name of [
		"claude-code-review.yml",
		"claude-code-review-run.yml",
		"claude-issue-agent.yml",
		"command-analyze.yml",
		"command-fix.yml",
	]) {
		const workflow = readYaml(`../../workflows/${name}`);
		assert.ok(Object.values(workflow.jobs).every((job) => !String(job.if).includes("false &&")));
	}
});
