# Coding Agent Automation

The shared action runs PR review, PR labeling, issue triage, `/analyze`, and
`/fix`. Workflows own their triggers, authorization, GitHub permissions, checkout,
status reporting and comment resolution. The action owns model selection, native
tool permissions, credentials, CLI setup and attribution.

## Model Selection

Set the repository **variable** `ACTIONS_MODEL` under
**Settings > Secrets and variables > Actions**:

| Value                                                                                  | Runner                                        |
| -------------------------------------------------------------------------------------- | --------------------------------------------- |
| Unset or empty                                                                         | Claude Code with its default model            |
| `opus`, `sonnet`, `haiku`, `opusplan`, or a `claude-*` model ID (also `[1m]` variants) | Claude Code with that model                   |
| `gpt-6-astra` or another `gpt-*` deployment                                            | OpenCode with that Azure deployment           |
| `azure/<deployment>`                                                                   | OpenCode with a custom-named Azure deployment |

Claude requires the existing `CLAUDE_CODE_OAUTH_TOKEN` secret. GPT requires only:

| Kind     | Name                    | Value                                         |
| -------- | ----------------------- | --------------------------------------------- |
| Variable | `AZURE_OPENAI_ENDPOINT` | `https://RESOURCE.openai.azure.com/openai/v1` |
| Secret   | `AZURE_OPENAI_API_KEY`  | API key for the Azure resource                |

`ACTIONS_MODEL` is also the Azure deployment name, so no separate deployment
setting is needed. The deployment must support the Responses API and tool
calling. OpenCode uses a conservative 128,000-token context budget and
16,384-token output budget. Missing credentials or an unknown model/profile fail
the action; they never silently select a different runner.

The original workflow names, reusable-workflow paths, concurrency groups and
`Claude Code Review` status context remain stable. No branch-protection rename is
required. The re-enabled workflows take effect after merging to the default
branch; later model switches require only changing `ACTIONS_MODEL`.

## Task Interface

Callers provide `profile` (`review`, `label`, `triage`, `analyze`, or `fix`),
`number`, optional `comment-id`, a trusted `prompt`, the scoped `github-token`, and
the two optional inference secrets. Only the selected runner receives its
inference credential. Native tool names and flags are private to the action.

Both runners use the same task prompts and command policy. Review and labeling
inspect trusted base code and remote metadata without editing or executing PR
code. Only triage and maintainer-invoked fix may edit and push. Unknown profiles
fail closed. These tool policies do not replace job token scopes or OS isolation;
build/test commands remain outside the current allowlist.

Both runners apply the same command guard before shell execution: only literal
single `git`/`gh` commands are accepted, shell operators and expansion are rejected,
Git read flags are restricted, and external diff/textconv and editor/browser
launches are disabled. Arguments are re-quoted so Markdown in comment bodies stays
literal. Write operations still require the profile's native tool permission.
Actor validation also runs before either runner; only reviews allow Dependabot.

The OpenCode hook removes its inference key from shell environments. Project
configuration, external skills, LSP, formatters and sharing are disabled for that
runner. Claude uses the existing pinned official action, with its available tools
and allowed commands constrained by the profile. The scoped GitHub token remains
available for permitted GitHub operations.

The action fails on execution errors; OpenCode also requires a completed turn in
its event stream. Completion is not an independent verification that a comment or
PR was published. Existing workflow lifecycle behavior is otherwise unchanged.

After installing the repository's Node dependencies, run
`npm ci --prefix .github/actions/coding-agent --ignore-scripts`,
`node --test .github/actions/coding-agent/*.test.mjs` and `actionlint`.
