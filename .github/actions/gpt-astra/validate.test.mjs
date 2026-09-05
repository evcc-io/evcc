import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { test } from "node:test";
import YAML from "yaml";
import scrubEnv from "./scrub-env.mjs";

const readYaml = (path) => YAML.parse(readFileSync(new URL(path, import.meta.url), "utf8"));
const action = readYaml("./action.yml");

test("shell steps parse", () => {
	for (const step of action.runs.steps) {
		const result = spawnSync("bash", ["-n"], { input: step.run, encoding: "utf8" });
		assert.equal(result.status, 0, result.stderr);
	}
});

test("missing or invalid Azure settings fail before installation", () => {
	const run = action.runs.steps[0].run;
	const env = {
		...process.env,
		ENDPOINT: "https://example.openai.azure.com/openai/v1",
		DEPLOYMENT: "astra-production",
		API_KEY: "test-only",
		PERMISSION: '{"*":"deny"}',
	};
	assert.equal(spawnSync("bash", ["-e", "-c", run], { env }).status, 0);
	for (const key of ["ENDPOINT", "DEPLOYMENT", "API_KEY", "PERMISSION"]) {
		assert.notEqual(spawnSync("bash", ["-e", "-c", run], { env: { ...env, [key]: "" } }).status, 0);
	}
	for (const endpoint of ["http://example/openai/v1", "https://example.openai.azure.com"]) {
		assert.notEqual(
			spawnSync("bash", ["-e", "-c", run], { env: { ...env, ENDPOINT: endpoint } }).status,
			0
		);
	}
	assert.notEqual(
		spawnSync("bash", ["-e", "-c", run], { env: { ...env, PERMISSION: '{"*":"allow"}' } }).status,
		0
	);
});

test("agent shells do not inherit the inference key", async () => {
	const hooks = await scrubEnv();
	const output = { env: { GH_TOKEN: "scoped-token" } };
	await hooks["shell.env"]({}, output);
	assert.equal(output.env.AZURE_OPENAI_API_KEY, "");
	assert.equal(output.env.ASTRA_PROMPT, "");
	assert.equal(output.env.GH_TOKEN, "scoped-token");
});

test("Azure endpoint normalization matches the bundled SDK", () => {
	const run = action.runs.steps.at(-1).run.split("git config")[0];
	for (const [endpoint, expected] of [
		["https://example.openai.azure.com/openai/v1", "https://example.openai.azure.com/openai"],
		["https://example.openai.azure.com/openai/v1/", "https://example.openai.azure.com/openai"],
		[
			"https://example.services.ai.azure.com/openai/v1",
			"https://example.services.ai.azure.com/openai/v1",
		],
	]) {
		const result = spawnSync("bash", ["-e", "-c", `${run}\nprintf '%s' "$AZURE_OPENAI_ENDPOINT"`], {
			env: { ...process.env, AZURE_OPENAI_ENDPOINT: endpoint },
			encoding: "utf8",
		});
		assert.equal(result.status, 0, result.stderr);
		assert.equal(result.stdout, expected);
	}
});

test("issue modes retain their write boundaries", () => {
	const workflow = readYaml("../../workflows/gpt-astra-issue-agent-run.yml");
	const step = workflow.jobs.run.steps.find((step) => step.uses === "./.github/actions/gpt-astra");
	for (const mode of ["triage", "analyze", "fix"]) {
		const rendered = step.with.permission.replace(/\$\{\{ (.*?) \}\}/g, (_match, expression) => {
			const condition = expression.match(/^inputs.mode == '(\w+)' && '(\w+)' \|\| '(\w+)'$/);
			if (condition) return mode === condition[1] ? condition[2] : condition[3];
			return {
				"inputs.issue_number": "123",
				"inputs.comment_id": "456",
				"github.repository": "evcc-io/evcc",
			}[expression];
		});
		const permission = JSON.parse(rendered);
		assert.equal(permission["*"], "deny");
		assert.equal(permission.bash["*"], "deny");
		assert.equal(permission.edit, mode === "analyze" ? "deny" : "allow");
		assert.equal(permission.bash["git push *"], mode === "analyze" ? "deny" : "allow");
		assert.equal(permission.bash["gh issue edit 123 *"], mode === "triage" ? "allow" : "deny");
	}
});

test("API errors and unfinished turns fail the action", () => {
	const run = action.runs.steps.at(-1).run;
	const filter = run.match(/jq -s -e '([^']+)'/)[1];
	for (const [events, expected] of [
		[[{ type: "step_finish", part: { reason: "stop" } }], 0],
		[[], 1],
		[[{ type: "step_finish", part: { reason: "length" } }], 1],
		[[{ type: "step_finish", part: { reason: "stop" } }, { type: "error" }], 1],
	]) {
		const input = events.map((event) => JSON.stringify(event)).join("\n");
		assert.equal(spawnSync("jq", ["-s", "-e", filter], { input }).status, expected);
	}
});
