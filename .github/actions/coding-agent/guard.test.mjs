import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { test } from "node:test";
import { normalizeCommand } from "./guard.mjs";
import scrubEnv from "./scrub-env.mjs";

const cli = (input) => {
	const result = spawnSync(
		process.execPath,
		[fileURLToPath(new URL("./guard.mjs", import.meta.url))],
		{
			input: JSON.stringify(input),
			encoding: "utf8",
		}
	);
	assert.equal(result.status, 0, result.stderr);
	assert.equal(result.stderr, "");
	return JSON.parse(result.stdout);
};

const rejected = [
	undefined,
	null,
	123,
	"",
	"   ",
	"git status\0",
	">file",
	"gh pr view 123 >file",
	"git status >>file",
	"git status 2>&1",
	"git status <file",
	"git status | cat",
	"git status && git diff",
	"git status; git diff",
	"git status &",
	"git status # comment",
	"git diff *.go",
	"PAGER=evil git log --oneline",
	"env git status",
	"/usr/bin/git status",
	"echo git status",
	"git",
	"git config x y",
	"git -c alias.log=evil log",
	"gh extension exec evil",
	"gh pr checkout 123",
	"git log $HOME",
	'git log "$HOME"',
	"git log ${HOME}",
	"git log $(id)",
	"git log <(id)",
	"git log ~/file",
	"git log '~'",
	'git log --out""put=file',
	'git log "--output" file',
	"git log --out\\put=file",
	"git log --out=file",
	"git diff --output file",
	"git diff --ext-diff",
	"git log --textconv",
	"git diff --no-index a b",
	"git log --unknown",
	"git log --max-count",
	"git log -n --output=file",
	"gh pr comment 123 --editor",
	'gh pr comment 123 --ed""itor',
	"gh pr comment 123 --ed\\itor",
	"gh pr view 123 --web",
	"gh pr view 123 --web=true",
	"gh pr comment 123 -e",
	"gh pr view 123 -w",
	"gh pr comment 123 -ve",
	"gh pr view 123 -vw",
	'gh pr view 123 -v""w',
];

for (const command of rejected) {
	test(`both hooks reject ${JSON.stringify(command)?.slice(0, 120)}`, async () => {
		assert.throws(() => normalizeCommand(command));
		const hooks = await scrubEnv();
		const output = { args: { command } };
		await assert.rejects(hooks["tool.execute.before"]({ tool: "bash" }, output));
		assert.equal(output.args.command, command);
		const result = cli({ tool_name: "Bash", tool_input: { command } }).hookSpecificOutput;
		assert.equal(result.hookEventName, "PreToolUse");
		assert.equal(result.permissionDecision, "deny");
		assert.equal(typeof result.permissionDecisionReason, "string");
		assert.ok(result.permissionDecisionReason.length);
		assert.equal(result.updatedInput, undefined);
	});
}

const markdown = "It's `git diff` > nothing\n\n```go\nx := 1\n```\n--editor -w $HOME $(id)";
const accepted = [
	["git status --short", ["git", "status", "--short"]],
	[
		'g""it lo\\g --oneline -n 5',
		["git", "log", "--no-ext-diff", "--no-textconv", "--oneline", "-n", "5"],
	],
	[
		"git log -10 --max-count=5 --format='%h %s' HEAD",
		[
			"git",
			"log",
			"--no-ext-diff",
			"--no-textconv",
			"-10",
			"--max-count=5",
			"--format=%h %s",
			"HEAD",
		],
	],
	[
		"git diff --stat --patch --name-only --name-status --check -U3 --cached",
		[
			"git",
			"diff",
			"--no-ext-diff",
			"--no-textconv",
			"--stat",
			"--patch",
			"--name-only",
			"--name-status",
			"--check",
			"-U3",
			"--cached",
		],
	],
	[
		"git diff HEAD -- --output --ext-diff --textconv",
		[
			"git",
			"diff",
			"--no-ext-diff",
			"--no-textconv",
			"HEAD",
			"--",
			"--output",
			"--ext-diff",
			"--textconv",
		],
	],
	[
		"git log --no-ext-diff --no-textconv",
		["git", "log", "--no-ext-diff", "--no-textconv", "--no-ext-diff", "--no-textconv"],
	],
	["gh pr view 123 --json title,body", ["gh", "pr", "view", "123", "--json", "title,body"]],
	["gh issue view 123", ["gh", "issue", "view", "123"]],
	["gh label list", ["gh", "label", "list"]],
	[
		"gh api repos/evcc-io/evcc/issues/comments/456 --jq .body",
		["gh", "api", "repos/evcc-io/evcc/issues/comments/456", "--jq", ".body"],
	],
	[
		'gh pr comment 123 --body "Use `git diff` > nothing\nnext line"',
		["gh", "pr", "comment", "123", "--body", "Use `git diff` > nothing\nnext line"],
	],
	[
		`gh issue comment 123 --body '${markdown.replaceAll("'", "'\\''")}'`,
		["gh", "issue", "comment", "123", "--body", markdown],
	],
	["gh issue comment 123 -b '-e'", ["gh", "issue", "comment", "123", "-b", "-e"]],
	["gh pr comment 123 --body='--web'", ["gh", "pr", "comment", "123", "--body=--web"]],
	["gh issue comment 123 --body ''", ["gh", "issue", "comment", "123", "--body", ""]],
	["git status\ngh pr view 123", ["git", "status", "gh", "pr", "view", "123"]],
];

for (const [command, argv] of accepted) {
	test(`both hooks canonicalize ${JSON.stringify(command)}`, async () => {
		const canonical = normalizeCommand(command);
		const original = {
			command,
			description: "Keep this field",
			timeout: 1234,
			run_in_background: false,
		};
		const hooks = await scrubEnv();
		const output = { args: { ...original } };
		await hooks["tool.execute.before"]({ tool: "bash" }, output);
		assert.deepEqual(output.args, { ...original, command: canonical });
		const result = cli({ tool_name: "Bash", tool_input: original });
		assert.deepEqual(result, {
			hookSpecificOutput: {
				hookEventName: "PreToolUse",
				updatedInput: { ...original, command: canonical },
			},
		});
		// Let a real shell decode argv without invoking git, gh or any network service.
		const shell = spawnSync(
			"bash",
			["--noprofile", "--norc", "-c", `set -- ${canonical}\nprintf '%s\\0' "$@"`],
			{
				env: { ...process.env, BASH_ENV: "" },
				encoding: "utf8",
			}
		);
		assert.equal(shell.status, 0, shell.stderr);
		assert.equal(shell.stderr, "");
		assert.deepEqual(shell.stdout.split("\0").slice(0, -1), argv);
	});
}

test("native permissions remain responsible for write profiles", () => {
	for (const command of [
		"git checkout -b fix",
		"git switch -c fix",
		"git add file",
		"git commit -m fix",
		"git push origin HEAD",
		"gh pr create --title fix --body fix",
		"gh issue edit 123 --add-label bug",
		"gh pr edit 123 --add-label bug",
	]) {
		assert.equal(normalizeCommand(command), command);
		assert.equal(
			cli({ tool_name: "Bash", tool_input: { command } }).hookSpecificOutput.permissionDecision,
			undefined
		);
	}
});

test("background requests are denied before execution", async () => {
	const hooks = await scrubEnv();
	for (const field of ["background", "run_in_background"]) {
		await assert.rejects(
			hooks["tool.execute.before"](
				{ tool: "bash" },
				{ args: { command: "git status", [field]: true } }
			),
			/Background/
		);
	}
	const result = cli({
		tool_name: "Bash",
		tool_input: { command: "git status", run_in_background: true },
	});
	assert.equal(result.hookSpecificOutput.permissionDecision, "deny");
});

test("other tools are untouched and malformed Bash input is denied", async () => {
	const hooks = await scrubEnv();
	const output = { args: { command: ">file" } };
	await hooks["tool.execute.before"]({ tool: "read" }, output);
	assert.deepEqual(output, { args: { command: ">file" } });
	assert.deepEqual(cli({ tool_name: "Read", tool_input: { file_path: "file" } }), {});
	await assert.rejects(hooks["tool.execute.before"]({ tool: "bash" }, {}));
	assert.equal(cli({ tool_name: "Bash" }).hookSpecificOutput.permissionDecision, "deny");
	const invalid = spawnSync(
		process.execPath,
		[fileURLToPath(new URL("./guard.mjs", import.meta.url))],
		{ input: "{", encoding: "utf8" }
	);
	assert.equal(invalid.status, 0);
	assert.equal(JSON.parse(invalid.stdout).hookSpecificOutput.permissionDecision, "deny");
});

test("shell environment scrubs secrets and neutralizes pagers", async () => {
	const hooks = await scrubEnv();
	const output = {
		env: {
			AZURE_OPENAI_API_KEY: "secret",
			AGENT_PROMPT: "prompt",
			GH_TOKEN: "scoped",
			GH_PAGER: "evil",
			PAGER: "evil",
			GIT_PAGER: "evil",
			GH_FORCE_TTY: "1",
		},
	};
	await hooks["shell.env"]({}, output);
	assert.deepEqual(output.env, {
		AZURE_OPENAI_API_KEY: "",
		AGENT_PROMPT: "",
		GH_TOKEN: "scoped",
		GH_PAGER: "cat",
		PAGER: "cat",
		GIT_PAGER: "cat",
		GH_FORCE_TTY: "",
	});
});
