#!/usr/bin/env python3
"""Unit tests for the tag-sync conflict classifier.

Run: python3 .github/scripts/test_resolve_tag_sync.py
No third-party dependencies (stdlib only), so it runs anywhere the sync does.

Convention (matches sync-tags.yml `git merge master` into the upstream tag):
  ours   = upstream release tag (HEAD)
  theirs = this fork (master)
"""

import sys

sys.path.insert(0, __file__.rsplit("/", 1)[0])
import resolve_tag_sync as R  # noqa: E402

PASS = 0
FAIL = 0


def check(name, got, want):
    global PASS, FAIL
    if got == want:
        PASS += 1
        print(f"ok   - {name}")
    else:
        FAIL += 1
        print(f"FAIL - {name}: got {got!r}, want {want!r}")


def wf(header, jobs):
    """Build a workflow doc from a header and a list of (name, body-lines)."""
    out = list(header) + ["jobs:"]
    for name, body in jobs:
        out.append(f"  {name}:")
        out.extend("    " + b for b in body)
    return "\n".join(out) + "\n"


HEADER = ["name: W", "on:", "  push:"]


def docker(runner="ubuntu-24.04", cleanup="fork"):
    step = "run: fork-cleanup" if cleanup == "fork" else "run: upstream-cleanup"
    return ("docker", [f"runs-on: {runner}", "steps:", f"- name: x", f"  {step}"])


# 1. runner-type only: shared job differs solely in runs-on -> benign
base = wf(HEADER, [docker(runner="depot-x")])
ours = wf(HEADER, [docker(runner="depot-arm")])   # upstream changes runner
theirs = wf(HEADER, [docker(runner="ubuntu-24.04")])  # fork changes runner
benign, _ = R.classify(base, ours, theirs)
check("runner-type only -> benign", benign, True)

# 2. removed job: fork deleted apt, upstream modified it -> benign
base = wf(HEADER, [docker(), ("apt", ["runs-on: depot-x", "run: old"])])
ours = wf(HEADER, [docker(), ("apt", ["runs-on: depot-x", "run: new-upstream"])])
theirs = wf(HEADER, [docker()])
benign, _ = R.classify(base, ours, theirs)
check("removed job (apt) -> benign", benign, True)

# 3. genuine change in a kept job -> needs review (PR)
base = wf(HEADER, [("docker", ["runs-on: ubuntu-24.04", "run: base"])])
ours = wf(HEADER, [("docker", ["runs-on: ubuntu-24.04", "run: upstream-change"])])
theirs = wf(HEADER, [("docker", ["runs-on: ubuntu-24.04", "run: fork-change"])])
benign, _ = R.classify(base, ours, theirs)
check("kept-job content conflict -> PR", benign, False)

# 4. brand-new upstream job (absent from base) conflicts with fork content and
#    would be dropped by the fork side -> PR. Both sides append a different job
#    at the same spot, forcing an add/add conflict.
base = wf(HEADER, [docker()])
ours = wf(HEADER, [docker(), ("newjob", ["runs-on: depot-x", "run: n"])])
theirs = wf(HEADER, [docker(), ("forkextra", ["runs-on: ubuntu-24.04", "run: f"])])
benign, _ = R.classify(base, ours, theirs)
check("new upstream job + conflict -> PR", benign, False)

# 5. combined benign: removed job AND runner-type change together -> benign
base = wf(HEADER, [docker(runner="depot-x"), ("hassio", ["runs-on: depot-x", "run: h"])])
ours = wf(HEADER, [docker(runner="depot-arm"), ("hassio", ["runs-on: depot-x", "run: h2"])])
theirs = wf(HEADER, [docker(runner="ubuntu-24.04")])
benign, _ = R.classify(base, ours, theirs)
check("removed job + runner change -> benign", benign, True)

# 6. header (top-level) conflict -> PR
base = wf(["name: W", "on:", "  push:"], [docker()])
ours = wf(["name: W", "on:", "  push:", "  schedule: upstream"], [docker()])
theirs = wf(["name: W", "on:", "  push:", "  schedule: fork"], [docker()])
benign, _ = R.classify(base, ours, theirs)
check("header conflict -> PR", benign, False)

print(f"\n{PASS} passed, {FAIL} failed")
sys.exit(1 if FAIL else 0)
