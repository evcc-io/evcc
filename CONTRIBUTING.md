# Contributing

## Developing

### Development environment

Developing evcc requires [Go][1] and [Node][2]. We recommend VSCode with the [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go), [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode) and [Vetur](https://marketplace.visualstudio.com/items?itemName=octref.vetur) extensions.

Alternatively, if you use VS Code and [devcontainers](https://code.visualstudio.com/docs/devcontainers/containers), you can use the "Dev containers: Clone repository in container volume" action. This will create a devcontainer with the required toolchain and install the prerequisites as explained below. Wait until the startup log says "Done. Press any key to close the terminal." and check for any errors.

We use linters (golangci-lint, Prettier) to keep a coherent source code formatting. It's recommended to use the format-on-save feature of your editor. You can manually reformat your code by running:

```sh
make lint
make lint-ui
```

### Device templates

The software supports a massive amount of different devices (charger, meter, vehicle, tariff) that are defined by **templates**.
A template can use the [plugin system](https://docs.evcc.io/docs/devices/plugins) (preferred) for communication with the device or reference a dedicated Go implementation.
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
npm install
npm run dev
```

### Storybook

We're using storybook to develop and visualize UI components in different states. Running the command below will open your browser at http://127.0.0.1:6006/.

```sh
npm run storybook
```

### Integration testing

We use Playwright for end-to-end integration tests. They start a local evcc instance with different configuration yamls and prefilled databases. To run them, you have to do a local build first.

```sh
make ui build
npm run playwright
```

### Simulating device state

Since we don't want to run tests against real devices or cloud services, we've build a simple simulator that lets you emulated meters, vehicles and loadpoints. The simulators web interface runs on http://localhost:7072.

```
npm run simulator
```

Run an evcc instance that uses simulator data. This configuration runs with a very high refresh interval to speed up testing.

```
make ui build
./evcc --config tests/simulator.evcc.yaml
```

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

[1]: https://go.dev
[2]: https://nodejs.org/

## Documentation, Website and iOS/Android App

We're always thankful for contributions.
Docs, website and app have dedicated repositories.
Please open a GitHub pull request in the respective repository.

- Documentation: [evcc-io/docs](https://github.com/evcc-io/docs)
- Website: [evcc-io/evcc.io](https://github.com/evcc-io/evcc.io)
- iOS/Android App: [evcc-io/app](https://github.com/evcc-io/app)

## License

By contributing to evcc, you agree that your contributions will be licensed under the existing license terms that apply to the respective parts of the project.
This constitutes an implicit Contributor License Agreement (CLA), following GitHub's standard practice where contributions are made under the same terms as the project license.
