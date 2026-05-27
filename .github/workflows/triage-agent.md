---
timeout-minutes: 5
strict: true
on:
  issues:
    types: [opened, edited, reopened]
  pull_request:
    types: [opened, edited, reopened]
if: "false"
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    toolsets: [issues, pull_requests, labels]
safe-outputs:
  add-labels:
    allowed: [bug, enhancement, documentation, question, device, tariff, vehicle, heating]
  add-comment: {}
source: githubnext/gh-aw/.github/workflows/issue-triage-agent.md@87fe98fa15e2bb50f41225a356bbc07318b54fcf
---

# Triage Agent

## Context

- **Repository**: ${{ github.repository }}

## Label the Issue/Pull Request

Look at the issue/pull request. Analyze title and body, then add one of the allowed labels: `bug`, `enhancement`, `documentation`, `question`, `device`, `tariff`, `vehicle`, `heating`.

Skip updating the issue/pull request if it already has a label attached.

If you add the `bug` label, also set the issue type to `bug`.

## Identify Supporters

If an issue is a `bug`, try to identify potential causes by looking at recent pull requests not older than 3 months.

If you find pull requests that may have introduced the bug, try identifying potential supporters for the issue. Supporters may be:

- authors or commentators of the pull request
- code owners for the code modified in the pull request (see CODEOWNERS file)

If you can identify a pull request that may have introduced the bug, mention the pull request in the issue. If identified, mention the supporter, explaining why he was mentioned.
