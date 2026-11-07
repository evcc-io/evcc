# Contributing

Thanks for your interest in evcc!
Most ways to help don't need any code: answering questions, testing new versions, improving support for your own devices, translating or editing the docs.
[docs.evcc.io/en/contribute](https://docs.evcc.io/en/contribute) gives an overview of all ways to help and of the project's repositories.
This file covers the ground rules and how to develop in this repository.

## Ground Rules

- **Start small**: Good first contributions come from your own setup, e.g. an inverter, battery, vehicle or charger that maintainers can't test. Don't start with your favourite big feature. Get to know the community first and iterate from there.
- **Discuss first**: Only open a pull request if there is a related issue or discussion where the team has indicated a positive tendency toward the proposed change. Link to it in your pull request description.
- **Keep it focused**: One topic per pull request, written in English (see [Communication Language](#communication-language)).
- **Understand your changes**: AI tools are welcome, but you remain responsible for your work regardless of which tools you used. Fully understand all changes and be prepared to answer questions about them.
- **Mark AI-generated text**: Issues, pull request descriptions and comments written by an AI tool must say so, e.g. with the tool's attribution footer. This lets readers decide how to treat them. Keep them clear and concise.
- **Respect review time**: Ensure contributions justify the review effort required from maintainers.

If a contribution doesn't seem to follow these rules, we'll close it.

General development topics are discussed on [Slack](https://evcc.io/slack).

## Developing

### Development environment

Developing evcc requires [Go][1] [Node][2] and [Vite+][3]. We recommend VSCode with the [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go), [Oxc](https://marketplace.visualstudio.com/items?itemName=oxc.oxc-vscode) and [Vue](https://marketplace.visualstudio.com/items?itemName=Vue.volar) extensions.

Alternatively, if you use VS Code and [devcontainers](https://code.visualstudio.com/docs/devcontainers/containers), you can use the "Dev containers: Clone repository in container volume" action. This will create a devcontainer with the required toolchain and install the prerequisites as explained below. Wait until the startup log says "Done. Press any key to close the terminal." and check for any errors.

We use linters (golangci-lint, oxlint/oxfmt via Vite+) to keep a coherent source code formatting. It's recommended to use the format-on-save feature of your editor. You can manually reformat your code by running:

```sh
make lint
make lint-ui
```

### Device templates

The software supports a massive amount of different devices (charger, meter, vehicle, tariff) that are defined by **templates**.
A template can use the [plugin system](https://docs.evcc.io/en/reference/plugins) (preferred) for communication with the device or reference a dedicated Go implementation.
All bundled templates are located in the [`/templates/definition`](https://github.com/evcc-io/evcc/tree/master/templates/definition) directory.

If you want to add a new plugin we recommend looking at existing, similar implementations for reference.
When your template requires Go code you have to build the project from source (see instructions below).
Otherwise you can use the evcc binary and point it to your new template file for testing.

```sh
evcc --template-type charger --template new-charger-template.yaml
```

Besides the actual device configuration, templates contain meta-data like product name, manufacturer, instructions how to configure the device to work with evcc.
On release, this data is extracted and pushed to the [`evcc-io/docs`](https://github.com/evcc-io/docs) repository to keep the documentation in sync. You can verify the generated meta-data by running:

```sh
make docs
```

This will write the documentation-relevant data to `/templates/docs`.

## Building from source

Install prerequisites (once):

```sh
make install-ui
make install
```

Build and run:

```sh
make
./evcc
```

Open UI at http://127.0.0.1:7070

To run without creating the `evcc` binary use:

    go run ./...

### Cross Compiling

To compile a version for an ARM device like a Raspberry Pi set GO command variables as needed, eg:

```sh
GOOS=linux GOARCH=arm GOARM=6 make
```

### Publishing docker images

```sh
make docker DOCKER_IMAGE=my/docker DOCKER_TAG=0815
```

## Debugging in VS Code

### evcc Core

To debug a local evcc build in VS Code, add the following entry to your `launch.json`.
You can adjust the referred configuration as needed to e.g. use your live configuration.

```json
{
    "name": "Launch evcc local build with demo config",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "${workspaceFolder}",
    "args": ["-c", "${workspaceFolder}/cmd/demo.yaml"],
    "cwd": "${workspaceFolder}",
},
```

### UI

For frontend development start the Vue toolchain in dev-mode. Open http://127.0.0.1:7071/ to get to the live reloading development server. It pulls its data from port 7070 (see above).

```sh
vp install
vp run dev
```

Start the backend with `--disable-auth` when checking the configuration UI of a
throw-away instance. Without it the UI asks for an administrator password first.

```sh
./evcc --config tests/config-with-tariffs.evcc.yaml --disable-auth
```

### Storybook

We're using storybook to develop and visualize UI components in different states. Running the command below will open your browser at http://127.0.0.1:6006/.

```sh
vp run storybook
```

### Integration testing

We use Playwright for end-to-end integration tests. They start a local evcc instance with different configuration yamls and prefilled databases. To run them, you have to do a local build first.

```sh
make ui build
vp run playwright
```

### Simulating device state

Since we don't want to run tests against real devices or cloud services, we've build a simple simulator that lets you emulated meters, vehicles and loadpoints. The simulators web interface runs on http://localhost:7072.

```
vp run simulator
```

Run an evcc instance that uses simulator data. This configuration runs with a very high refresh interval to speed up testing.

```
make ui build
./evcc --config tests/simulator.evcc.yaml
```

### Test builds

Maintainers can comment `/build` on a pull request.
This publishes binaries and a Docker image `evcc/evcc:pr-<number>`, so others can test the change on their own hardware.

## Communication Language

evcc has a large German-speaking user base, but we want to be open and accessible to everyone in the global community. To balance these needs:

- **Pull Requests**
  - 🇬🇧 English required
- **Issues**
  - 🇬🇧 English recommended
  - 🇩🇪 German acceptable to start, must switch to English after first English comment
- **GitHub Discussions**
  - 🇬🇧 🇩🇪 Both English and German allowed

💬 _Non-German speakers: We strongly encourage you to ask participants to switch to English. For pull requests, we have a language check bot that does this automatically._

Thank you all for helping make evcc accessible! 🌍

## Adding or modifying translations

evcc already includes many translations for the UI. We're using [Weblate](https://hosted.weblate.org/projects/evcc/evcc/) to maintain translations. Feel free to add more languages or verify and edit existing translations. Weblate will automatically push all modifications to the evcc repository where they get reviewed and merged.

If you find a text that is not yet translatable in [Weblate](https://hosted.weblate.org/projects/evcc/evcc/), you can help us by making it translatable. To do this, you can simply find the missing translation text in the code and apply similar changes as in these two Pull Requests:

- [UI: Add missing translation for Error during startup](https://github.com/evcc-io/evcc/pull/14695)
- [Translation: kein Plan, keine Grenze](https://github.com/evcc-io/evcc/pull/7461/)

Note: To ensure the build succeeds after creating new translations, make sure to include your new translations in both the [de.json](i18n/de.json) and [en.json](i18n/en.json) files.

[![Languages](https://hosted.weblate.org/widgets/evcc/-/evcc/multi-auto.svg)](https://hosted.weblate.org/engage/evcc/)

## Other Repositories

Docs, website, app, optimizer and Linux images have dedicated repositories.
See the [project overview](https://docs.evcc.io/en/contribute#projects) and open pull requests in the respective repository.

## License

By contributing to evcc, you agree that your contributions will be licensed under the existing license terms that apply to the respective parts of the project.
This constitutes an implicit Contributor License Agreement (CLA), following GitHub's standard practice where contributions are made under the same terms as the project license.

## Releases

Releases are cut by pushing a `MAJOR.MINOR.PATCH` tag. Any other tag is ignored.

Feature releases (`0.313.0`) must be tagged on `master`. The release workflow
rejects a feature tag that is not reachable from `master`.

Bugfix releases (`0.313.1`) may be tagged on any branch. That allows servicing
an older release line without shipping everything that has landed on `master`
since. Only the newest release moves the `latest` pointers, so a bugfix release
of an older line publishes its artifacts, but leaves the `evcc/evcc:latest`
docker tag, the homebrew formula, the GitHub latest release, the hassio addon
and the demo instance untouched.

To move a merged pull request onto a release branch, comment `/backport` on it.
The pull request has to carry the `bug` label and must not be marked `(BC)`,
only non-breaking bugfixes are backported.
The commit is cherry-picked onto the branch of the next bugfix release, e.g.
`release/0.313.1`, and a pull request is opened against it. The branch is
created at the newest tag of that line if it does not exist yet. Pass a branch
name, `/backport release/0.312.2`, to service an older line.

Releasing a bugfix deletes its release branch, since the tag is the branch tip
and everything on it has shipped. A branch that received further backports after
the tag is kept.

[1]: https://go.dev
[2]: https://nodejs.org/
[3]: https://viteplus.dev/guide/#install-vp
