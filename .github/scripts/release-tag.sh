#!/usr/bin/env bash
# Validates a release tag and reports whether it is the newest release.
#
# Feature releases (patch level 0) must be tagged on master. Bugfix releases may
# be tagged on any branch so an older release line can be serviced without
# shipping everything that landed on master since.
#
# Prints `latest=<true|false>` for consumption as a GitHub step output. Only the
# newest release may move the `latest` pointers (docker tag, homebrew formula,
# GitHub latest release, hassio addon, demo instance).
#
# Expects a checkout with `fetch-depth: 0`, which populates both the remote
# tracking branches and all tags.
#
# Run `release-tag.sh --self-test` to exercise the logic in a scratch repository.

set -euo pipefail

# overridden with the repository default branch by the workflow
MASTER_REF="${MASTER_REF:-origin/master}"

validate() {
	local tag=$1

	if [[ ! $tag =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
		echo "::error::invalid release tag '$tag', expected MAJOR.MINOR.PATCH" >&2
		return 1
	fi

	if [[ ${BASH_REMATCH[3]} == 0 ]] && ! git merge-base --is-ancestor "$tag" "$MASTER_REF"; then
		echo "::error::feature release '$tag' must be tagged on master" >&2
		return 1
	fi

	local newest
	newest=$(git tag --list | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort --version-sort | tail -1) || true

	if [[ $newest == "$tag" ]]; then
		echo "latest=true"
	else
		echo "latest=false"
	fi
}

self_test() {
	dir=$(mktemp -d)
	trap 'rm -rf "$dir"' EXIT
	cd "$dir"

	commit() { git -c user.email=t@t -c user.name=t commit --quiet --allow-empty -m "$1"; }

	git init --quiet --initial-branch=master .
	commit one
	git tag 0.1.0
	git checkout --quiet -b fix
	commit two
	git tag 0.1.1 # bugfix release off master
	git tag 0.2.0 # feature release off master
	git checkout --quiet master
	commit three
	git tag 0.3.0
	git tag 9.9.9-fork # tags from forks must not count as a release

	MASTER_REF=master

	failed=0
	expect() { # expect <tag> <expected output or FAIL>
		local got
		if ! got=$(validate "$1" 2>/dev/null); then got=FAIL; fi
		if [[ $got != "$2" ]]; then
			echo "FAIL: $1 -> $got, want $2" >&2
			failed=1
		fi
	}

	expect 0.1.0 latest=false # feature release on master, superseded
	expect 0.1.1 latest=false # bugfix release off master, older line
	expect 0.3.0 latest=true  # newest release
	expect 0.2.0 FAIL         # feature release not on master
	expect 0.1 FAIL           # not MAJOR.MINOR.PATCH
	expect v0.1.0 FAIL        # no v prefix allowed

	[[ $failed == 0 ]] && echo "self-test ok"
	return $failed
}

if [[ ${1:-} == --self-test ]]; then
	self_test
else
	validate "${1:?usage: release-tag.sh <tag>}"
fi
