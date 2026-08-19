#!/usr/bin/env python3
"""Auto-resolve the recurring, mechanical conflicts produced by the daily tag
sync (see .github/workflows/sync-tags.yml), which merges `master` (this fork)
into each new upstream release tag.

The fork permanently diverges from upstream in a few well-understood ways, and
those divergences generate the same conflicts on almost every sync:

  1. Deleted workflows  - the fork removes some upstream workflow files
                          (Claude review, documentation). Kept deleted.
  2. Removed jobs       - the fork only builds Docker, so it removes jobs such
                          as `apt`, the Home Assistant add-on (`hassio`) and the
                          fly.io `demo` job. Kept removed.
  3. Runner type        - upstream uses `depot-*` runners; the fork uses the
                          GitHub-hosted runners. Keep the fork's runner.

Any *other* conflict (a real change to a job the fork keeps, a header/`on:`
change, a brand-new upstream job that conflicts, etc.) is left unresolved so the
sync workflow opens a pull request for manual review.

Only files under .github/workflows/ are touched; conflicts anywhere else are
always left for a pull request.

The script operates on the in-progress merge's index stages:
  :1 = merge base, :2 = ours (HEAD = upstream tag), :3 = theirs (master = fork).

Exit code is always 0; the caller decides commit-vs-PR from the remaining
unmerged paths.
"""

import os
import re
import subprocess
import sys
import tempfile

WORKFLOW_PREFIX = ".github/workflows/"
JOB_KEY = re.compile(r"^  ([A-Za-z0-9_.-]+):\s*(#.*)?$")
RUNS_ON = re.compile(r"^\s*runs-on:")


def git(*args, check=True):
    return subprocess.run(
        ["git", *args], capture_output=True, text=True, check=check
    )


def stage_blob(stage: int, path: str):
    """Return the text of an index stage, or None if that stage is absent
    (e.g. no merge base, or the file is deleted on one side)."""
    r = git("show", f":{stage}:{path}", check=False)
    return r.stdout if r.returncode == 0 else None


def merge_file(ours: str, base: str, theirs: str, favor: str) -> str:
    """Three-way merge favoring one side on conflicts, keeping cleanly-merged
    content everywhere else. favor is 'ours' (upstream) or 'theirs' (fork)."""
    fds = []
    try:
        paths = []
        for content in (ours, base, theirs):
            fd, p = tempfile.mkstemp(suffix=".yml")
            os.write(fd, (content or "").encode())
            os.close(fd)
            paths.append(p)
            fds.append(p)
        # git merge-file <current> <base> <other>; current=ours, other=theirs
        r = subprocess.run(
            ["git", "merge-file", "-p", f"--{favor}", paths[0], paths[1], paths[2]],
            capture_output=True,
            text=True,
        )
        return r.stdout
    finally:
        for p in fds:
            try:
                os.unlink(p)
            except OSError:
                pass


def parse_workflow(text: str):
    """Split a workflow into (header_lines, {job_name: block_lines}).

    header = everything up to and including the top-level `jobs:` line;
    each job block spans from its 2-space-indented key to the next one."""
    lines = (text or "").split("\n")
    header = []
    jobs = {}
    order = []
    i = 0
    # header up to and including the `jobs:` line
    while i < len(lines):
        header.append(lines[i])
        if lines[i].rstrip() == "jobs:":
            i += 1
            break
        i += 1
    cur = None
    for line in lines[i:]:
        m = JOB_KEY.match(line)
        if m:
            cur = m.group(1)
            jobs[cur] = []
            order.append(cur)
        if cur is None:
            # Only comments/blanks legitimately sit between `jobs:` and the first
            # job. Ignore anything else: a conflict that straddles a commented-out
            # job (e.g. the fork disables check_date while upstream edits it) can
            # leave orphaned job-body lines here that must not skew the compare.
            continue
        jobs[cur].append(line)
    return header, jobs, order


def canon(lines):
    """Canonical form for *comparison only*: ignore blank lines, full-line
    comments and trailing whitespace. This lets a job the fork disabled by
    commenting it out (e.g. check_date) compare equal to its absence, and
    ignores pure formatting differences. The resolution text keeps comments."""
    out = []
    for ln in lines:
        s = ln.rstrip()
        if s == "" or s.lstrip().startswith("#"):
            continue
        out.append(s)
    return out


def strip_runs_on(lines):
    return [ln for ln in lines if not RUNS_ON.match(ln)]


def job_names(text):
    return set(parse_workflow(text)[1].keys())


def classify(base, ours, theirs):
    """Decide whether a workflow content conflict is confined to the fork's
    known divergences. Returns (benign, resolution_text)."""
    fork = merge_file(ours, base, theirs, "theirs")  # fork wins conflicts
    upstream = merge_file(ours, base, theirs, "ours")  # upstream wins conflicts

    if canon(fork.split("\n")) == canon(upstream.split("\n")):
        return True, fork  # conflict was cosmetic (whitespace/comments only)

    fh, fj, _ = parse_workflow(fork)
    uh, uj, _ = parse_workflow(upstream)

    # header (name/on/permissions/...) must not differ -> otherwise real conflict
    if canon(fh) != canon(uh):
        return False, None

    base_jobs = job_names(base) if base is not None else set()
    theirs_jobs = job_names(theirs) if theirs is not None else set()
    # jobs the fork deliberately removed: existed in base, gone from the fork
    fork_removed = base_jobs - theirs_jobs

    dropped = set(uj) - set(fj)  # jobs lost by choosing the fork's side
    if not dropped.issubset(fork_removed):
        return False, None  # dropping a job the fork didn't remove -> review

    if set(fj) - set(uj):
        return False, None  # unexpected fork-only job appeared on conflict

    # shared jobs may differ ONLY by runner type
    for name in set(fj) & set(uj):
        if canon(strip_runs_on(fj[name])) != canon(strip_runs_on(uj[name])):
            return False, None

    return True, fork


def main():
    unmerged = git("diff", "--name-only", "--diff-filter=U").stdout.split()
    for path in unmerged:
        if not path.startswith(WORKFLOW_PREFIX):
            print(f"leave (not a workflow): {path}")
            continue

        # modify/delete: file removed on master (theirs) -> honor the deletion
        if git("cat-file", "-e", f"master:{path}", check=False).returncode != 0:
            git("rm", "-q", "--", path)
            print(f"resolved (kept deletion): {path}")
            continue

        base = stage_blob(1, path)
        ours = stage_blob(2, path)
        theirs = stage_blob(3, path)
        if ours is None or theirs is None:
            print(f"leave (add/delete conflict): {path}")
            continue

        benign, resolution = classify(base, ours, theirs)
        if benign:
            with open(path, "w") as f:
                f.write(resolution)
            git("add", "--", path)
            print(f"resolved (fork divergence only): {path}")
        else:
            print(f"leave (needs review): {path}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
