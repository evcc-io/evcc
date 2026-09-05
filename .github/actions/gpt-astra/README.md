# GPT-Astra Automation

The PR reviewer, PR labeler, issue triage agent, `/analyze`, and `/fix` use
OpenCode with a GPT-Astra deployment on Azure OpenAI.

Configure these under **Settings > Secrets and variables > Actions**:

| Kind     | Name                      | Value                                         |
| -------- | ------------------------- | --------------------------------------------- |
| Variable | `AZURE_OPENAI_ENDPOINT`   | `https://RESOURCE.openai.azure.com/openai/v1` |
| Variable | `AZURE_OPENAI_DEPLOYMENT` | Your GPT-Astra deployment name                |
| Secret   | `AZURE_OPENAI_API_KEY`    | An API key for that Azure resource            |

The deployment must support the Responses API and tool calling. The runner uses
a conservative 128,000-token context budget and 16,384-token output budget.
The deployment name is independent of OpenCode's local `azure/gpt-6-astra` alias.
There is no fallback to a public provider or to Claude if configuration is missing.

The migration must reach the default branch before privileged workflows use it.
Replace any required `Claude Code Review` branch-protection check with
`GPT-Astra Code Review` when deploying it. This migration does not change the
separately configured Alibaba Open Code Review or Sourcery integrations.

PR workflows only check out trusted base code. The review and label agents cannot
edit files, run builds, or check out PR heads. Each caller supplies a deny-by-default
tool policy; only triage and maintainer-invoked `/fix` may edit and push fixes.
The inference key is removed from agent shell environments, while the scoped
GitHub token remains available for permitted `gh` commands. Project-local agent
configuration, external skills, LSP servers, and automatic sharing are disabled.

OpenCode is version-pinned in `action.yml`. Its JSON event stream must include a
completed turn and no API/session error for the action to succeed.

After installing the repository's Node dependencies, validate the runner with
`node --test .github/actions/gpt-astra/validate.test.mjs` and lint the workflows
with `actionlint`.
