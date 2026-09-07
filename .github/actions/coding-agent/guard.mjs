import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import shellQuote from "shell-quote";

const readFlags = new Set([
	"--oneline",
	"-p",
	"--patch",
	"-s",
	"--no-patch",
	"-u",
	"--stat",
	"--shortstat",
	"--numstat",
	"--name-only",
	"--name-status",
	"--check",
	"--summary",
	"--raw",
	"--no-ext-diff",
	"--no-textconv",
	"--no-color",
	"--no-renames",
	"--full-index",
	"--binary",
	"--cached",
	"--staged",
	"--exit-code",
	"--quiet",
	"--relative",
	"--all",
	"--first-parent",
	"--no-merges",
	"--merges",
	"--reverse",
	"--topo-order",
	"--date-order",
	"--graph",
	"--decorate",
	"--no-decorate",
	"--follow",
	"--abbrev-commit",
	"--no-abbrev-commit",
	"--full-history",
	"--simplify-merges",
	"--root",
	"-w",
	"--ignore-all-space",
	"-b",
	"--ignore-space-change",
	"--ignore-space-at-eol",
	"--ignore-blank-lines",
	"--minimal",
	"--patience",
	"--histogram",
]);
const readValueFlags = new Set([
	"-n",
	"--max-count",
	"--skip",
	"--since",
	"--until",
	"--after",
	"--before",
	"--author",
	"--committer",
	"--grep",
	"--format",
	"--pretty",
	"--date",
	"--diff-filter",
	"--unified",
	"--abbrev",
]);

export function normalizeCommand(command) {
	if (typeof command !== "string" || !command.trim() || command.includes("\0")) {
		throw new Error("Expected a nonempty shell command without NUL bytes");
	}
	const argv = shellQuote.parse(command, () => {
		throw new Error("Shell expansion is not allowed");
	});
	if (argv.some((arg) => typeof arg !== "string")) {
		throw new Error("Shell operators, redirects, globs and comments are not allowed");
	}
	// shell-quote does not escape a leading tilde when emitting a bare word.
	if (argv.some((arg) => arg.startsWith("~"))) throw new Error("Tilde expansion is not allowed");
	const [program, subcommand] = argv;
	if (program === "git") {
		if (
			!["log", "diff", "status", "checkout", "switch", "add", "commit", "push"].includes(subcommand)
		) {
			throw new Error("Unsupported git subcommand");
		}
		if (["log", "diff"].includes(subcommand)) {
			for (let i = 2; i < argv.length && argv[i] !== "--"; i++) {
				const arg = argv[i];
				if (!arg.startsWith("-") || readFlags.has(arg)) continue;
				const flag = arg.split("=", 1)[0];
				if (readValueFlags.has(flag)) {
					if (!arg.includes("=") && (!argv[++i] || argv[i].startsWith("-"))) {
						throw new Error(`Missing value for ${flag}`);
					}
					continue;
				}
				if (/^-(?:\d+|n\d+|U\d+|[MC](?:\d+%?)?)$/.test(arg)) continue;
				throw new Error(`Unsupported git read flag: ${arg}`);
			}
			// Override trusted config too, not just explicit command-line helpers.
			argv.splice(2, 0, "--no-ext-diff", "--no-textconv");
		}
	} else if (program === "gh") {
		const commands = {
			pr: ["view", "diff", "comment", "edit", "create"],
			issue: ["view", "comment", "edit"],
			label: ["list"],
		};
		if (subcommand !== "api" && !commands[subcommand]?.includes(argv[2])) {
			throw new Error("Unsupported gh subcommand");
		}
		for (let i = subcommand === "api" ? 2 : 3; i < argv.length && argv[i] !== "--"; i++) {
			const arg = argv[i];
			if (/^--(?:editor|web)(?:=|$)/.test(arg) || /^-[^-]*[ew]/.test(arg)) {
				throw new Error("gh editor and web flags are not allowed");
			}
			if (["--body", "-b", "--title", "-t"].includes(arg)) i++;
		}
	} else {
		throw new Error("Only literal git and gh commands are allowed");
	}
	// Never execute the original source: quoted backticks and newlines stay literal.
	return shellQuote.quote(argv);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
	let hookSpecificOutput;
	try {
		const input = JSON.parse(readFileSync(0, "utf8"));
		if (input.tool_name === "Bash") {
			if (input.tool_input?.run_in_background)
				throw new Error("Background commands are not allowed");
			const command = normalizeCommand(input.tool_input?.command);
			hookSpecificOutput = {
				hookEventName: "PreToolUse",
				updatedInput: { ...input.tool_input, command },
			};
		}
	} catch (error) {
		hookSpecificOutput = {
			hookEventName: "PreToolUse",
			permissionDecision: "deny",
			permissionDecisionReason: error.message,
		};
	}
	console.log(JSON.stringify(hookSpecificOutput ? { hookSpecificOutput } : {}));
}
