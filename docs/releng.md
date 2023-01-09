# Branching concept

- `master` is the default branch
- releases `SHOULD` be triggered from `master` branch
- release will trigger when tag is pushed on any branch
- to create bugfix release, branch from `master` and create a tag
- releases `SHOULD` not be created locally as they will conflict with pushing tagged branch

## Contents

Release contains:

- Github
- Docker
- Cloudsmith
