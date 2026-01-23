---
timeout-minutes: 5
strict: true
on:
  schedule: "0 14 * * 1-5"
  workflow_dispatch:
permissions:
  issues: read
tools:
  github:
    toolsets: [issues, labels]
safe-outputs:
  add-labels:
    allowed: [bug, feature, enhancement, documentation, question, help-wanted, good-first-issue]
  add-comment: {}
source: githubnext/gh-aw/.github/workflows/issue-triage-agent.md@87fe98fa15e2bb50f41225a356bbc07318b54fcf
---

# Issue Triage Agent

List open issues in ${{ github.repository }} that have no labels. For each unlabeled issue, analyze the title and body, then add one of the allowed labels: `bug`, `feature`, `enhancement`, `documentation`, `question`, `help-wanted`, or `good-first-issue`. 

Skip issues that:
- Already have any of these labels
- Have been assigned to any user (especially non-bot users)

After adding the label to an issue, mention the issue author in a comment explaining why the label was added.
