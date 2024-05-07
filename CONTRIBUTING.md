## Contributing

To build evcc from source, [Go][1] 1.22 and [Node][2] 18 are required.

Build and run go backend. The UI becomes available at http://127.0.0.1:7070/

```sh
make install-ui
make ui
make install
make
./evcc
```

### Backend development (VS Code)

The `launch.json`-configurations listed below are available in [`.vscode/example.launch.json`](./.vscode/example.launch.json). Instructions for how to deploy them are in that file, too.

After adding the configuration(s) to the actual `launch.json`, you can start the debugger from VS Code (default <kbd>F5</kbd>).

#### Available Debug Configurations in example.launch.json:

`"Debug local evcc demo"`:  
Run a local instance of evcc (localhost:7070). You can adjust the referred yaml-configuration (default `cmd/demo.yaml`) to e.g. use your live configuration.

`"Debug decorator"`:  
This can be used for specifically debugging the decorator.

### Cross Compile

To compile a version for an ARM device like a Raspberry Pi set GO command variables as needed, eg:

```sh
GOOS=linux GOARCH=arm GOARM=6 make
```

### UI development

For frontend development start the Vue toolchain in dev-mode. Open http://127.0.0.1:7071/ to get to the livelreloading development server. It pulls its data from port 7070 (see above).

```sh
npm install
npm run dev
```

### Integration tests

We use Playwright for end-to-end integration tests. They start a local evcc instance with different configuration yamls and prefilled databases. To run them, you have to do a local build first.

```sh
make ui build
npm run playwright
```

#### Simulating device state

Since we don't want to run tests against real devices or cloud services, we've build a simple simulator that lets you emulated meters, vehicles and loadpoints. The simulators web interface runs on http://localhost:7072.

```
npm run simulator
```

Run an evcc instance that uses simulator data. This configuration runs with a very high refresh interval to speed up testing.

```
make ui build
./evcc --config tests/simulator.evcc.yaml
```

### Code formatting

We use linters (golangci-lint, Prettier) to keep a coherent source code formatting. It's recommended to use the format-on-save feature of your editor. For VSCode use the [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go), [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode) and [Vetur](https://marketplace.visualstudio.com/items?itemName=octref.vetur) extension. You can manually reformat your code by running:

```sh
make lint
make lint-ui
```

### Publishing docker images

```sh
make docker DOCKER_IMAGE=my/docker DOCKER_TAG=0815
```

### Changing templates

evcc supports a massive amount of different devices. To keep our documentation and website in sync with the latest software the core project (this repo) generates meta-data that's pushed to the `docs` and `evcc.io` repository. Make sure to update this meta-data every time you make changes to a templates.

```sh
make docs
```

If you miss one of the above steps Gitub Actions will likely trigger a **Porcelain** error.

### Adding or modifying translations

evcc already includes many translations for the UI. Weblate Hosted is used to maintain all languages. Feel free to add more languages or verify and edit existing translations. Weblate will automatically push all modifications on a regular base to the evcc repository.

[![Weblate Hosted](https://hosted.weblate.org/widgets/evcc/-/evcc/287x66-grey.png)](https://hosted.weblate.org/engage/evcc/)
[![Languages](https://hosted.weblate.org/widgets/evcc/-/evcc/multi-auto.svg)](https://hosted.weblate.org/engage/evcc/)

https://hosted.weblate.org/projects/evcc/evcc/

[1]: https://go.dev
[2]: https://nodejs.org/
