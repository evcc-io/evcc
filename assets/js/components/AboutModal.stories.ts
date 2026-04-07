import AboutModal from "./AboutModal.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

const releaseNotes = `
<h1>v0.303.1</h1>
<h2>Changelog</h2>
<h3>Other Changes ☀️</h3>
<ul>
<li>Home Assistant: allow switch for enable/disable</li>
<li>Nexblue: remove broken 1p3p</li>
<li>Optimizer: return infeasable error</li>
<li>Revert: Safari: web socket bug redirect workaround</li>
</ul>
<h3>Bug Fixes 🐞</h3>
<ul>
<li>HomeAssistant: fix changelog</li>
<li>Optimizer: fix invalid battery capacity</li>
<li>SGready: fix panic</li>
</ul>
<h1>v0.303.0</h1>
<h2>Changelog</h2>
<h3>Breaking Changes 🚨</h3>
<ul>
<li>HEMS: refactor handling of production/feedin limits (BC)</li>
<li>Migrate optimizer (BC)</li>
</ul>
<h3>New Features 💫</h3>
<ul>
<li>Add RAEDIAN NEO and NEX AC charger</li>
<li>Audi: add vehicle features</li>
<li>ChargeX: add heartbeat to prevent PAC_Target_Timeout fallback</li>
<li>Heating: add continuous feature to improve heatpump experience</li>
<li>Sigenergy: add maxacpower</li>
<li>Tariff UI: add multiline formula support</li>
</ul>
<h3>Bug Fixes 🐞</h3>
<ul>
<li>Mercedes: fix auth</li>
<li>Nexblue: fix phase switching API endpoint</li>
<li>Planner: fix <code>this.updatePlanPreviewDebounced is null</code></li>
<li>Websocket: fix logging breaking digest auth</li>
</ul>
`;

export default {
  title: "AboutModal",
  component: AboutModal,
  argTypes: {
    installed: { control: "text" },
    commit: { control: "text" },
    availableVersion: { control: "text" },
    releaseNotes: { control: "text" },
    hasUpdater: { control: "boolean" },
    uploadMessage: { control: "text" },
    uploadProgress: { control: "number" },
  },
} as Meta<typeof AboutModal>;

const Template: StoryFn<typeof AboutModal> = (args) => ({
  components: { AboutModal },
  setup() {
    return { args };
  },
  template: '<AboutModal v-bind="args" />',
  mounted() {
    const el = document.getElementById("aboutModal");
    if (el) {
      import("bootstrap/js/dist/modal").then(({ default: Modal }) => {
        Modal.getOrCreateInstance(el).show();
      });
    }
  },
});

export const Stable = Template.bind({});
Stable.args = {
  installed: "0.303.1",
  availableVersion: "0.303.1",
};

export const StableUpdateAvailable = Template.bind({});
StableUpdateAvailable.args = {
  installed: "0.303.0",
  availableVersion: "0.303.1",
  releaseNotes,
};

export const StableUpdateWithUpdater = Template.bind({});
StableUpdateWithUpdater.args = {
  installed: "0.303.0",
  availableVersion: "0.303.1",
  releaseNotes,
  hasUpdater: true,
};

export const Nightly = Template.bind({});
Nightly.args = {
  installed: "0.303.1",
  commit: "5ce7be4a9f3b2c1d",
  availableVersion: "0.303.1",
};

export const DevBuild = Template.bind({});
DevBuild.args = {
  installed: "0.0.0",
};
