import Version from "./Version.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

const releaseNotes =
  '<h1>0.141</h1><p>Bug fixes:</p><ul><li>Withdraw Kia and Hyundai apis (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="817450535" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/699" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/699/hovercard" href="https://github.com/evcc-io/evcc/pull/699">#699</a>)</li><li>Simplify Tesla integration and fix upstream changes (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="817525274" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/700" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/700/hovercard" href="https://github.com/evcc-io/evcc/pull/700">#700</a>)</li><li>SHM: Check connected status (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="811567860" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/673" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/673/hovercard" href="https://github.com/evcc-io/evcc/pull/673">#673</a>)</li></ul><p>Enhancements:</p><ul><li>Add Seat api (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="812885392" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/681" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/681/hovercard" href="https://github.com/evcc-io/evcc/pull/681">#681</a>)</li><li>Add Skoda api (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="812883796" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/680" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/680/hovercard" href="https://github.com/evcc-io/evcc/pull/680">#680</a>)</li><li>Add Peugeot/Opel/Citroen api (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="815612446" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/692" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/692/hovercard" href="https://github.com/evcc-io/evcc/pull/692">#692</a>)</li><li>KEBA: Support mA current resolution adjustments (<a class="issue-link js-issue-link" data-error-text="Failed to load title" data-id="801358501" data-permission-text="Title is private" data-url="https://github.com/evcc-io/evcc/issues/646" data-hovercard-type="pull_request" data-hovercard-url="/evcc-io/evcc/pull/646/hovercard" href="https://github.com/evcc-io/evcc/pull/646">#646</a>)</li></ul>';

export default {
  title: "Footer/Version",
  component: Version,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    installed: { control: "text" },
    available: { control: "text" },
    commit: { control: "text" },
    releaseNotes: { control: "text" },
    hasUpdater: { control: "boolean" },
  },
} as Meta<typeof Version>;

const Template: StoryFn<typeof Version> = (args) => ({
  components: { Version },
  setup() {
    return { args };
  },
  template: '<Version v-bind="args" />',
});

export const Latest = Template.bind({});
Latest.args = {
  installed: "0.141",
  available: "0.141",
};

export const Nightly = Template.bind({});
Nightly.args = {
  installed: "0.141",
  available: "0.141",
  commit: "5ce7be4",
};

export const UpdateAvailable = Template.bind({});
UpdateAvailable.args = {
  installed: "0.141",
  available: "0.142",
  releaseNotes,
};

export const AutoUpdater = Template.bind({});
AutoUpdater.args = {
  installed: "0.141",
  available: "0.142",
  releaseNotes,
  hasUpdater: true,
};

export const NoReleaseNotes = Template.bind({});
NoReleaseNotes.args = {
  installed: "0.141",
  available: "0.142",
};
