import SponsorTokenRequired from "./SponsorTokenRequired.vue";
import api from "@/api";
import type { Meta, StoryFn } from "@storybook/vue3";
import type { AxiosAdapter } from "axios";
import { userEvent, within } from "storybook/test";

export default {
  title: "Config/SponsorTokenRequired",
  component: SponsorTokenRequired,
  globals: { backgrounds: { value: "box" } },
  parameters: {
    docs: {
      description: {
        component:
          "Inline sponsor token entry shown in device and feature modals for sponsor-only templates. The token is applied in place, the sponsor status arrives over the websocket.",
      },
    },
  },
} as Meta<typeof SponsorTokenRequired>;

// stories answer the token request themselves instead of hitting a backend
function respondWith(adapter: AxiosAdapter) {
  api.interceptors.request.use((config) => ({ ...config, adapter }));
}

const Template: StoryFn<typeof SponsorTokenRequired> = (args) => ({
  components: { SponsorTokenRequired },
  setup() {
    return { args };
  },
  template: '<div style="max-width: 560px"><SponsorTokenRequired v-bind="args" /></div>',
});

async function typeAndApply(canvasElement: HTMLElement) {
  const canvas = within(canvasElement);
  await userEvent.type(canvas.getByRole("textbox"), "eyJhbGciOiJIUzI1NiJ9.sponsor.token");
  await userEvent.click(canvas.getByRole("button", { name: "Apply" }));
}

export const Device = Template.bind({});
Device.args = {};

export const Feature = Template.bind({});
Feature.args = { feature: true };

export const Saving = Template.bind({});
Saving.args = {};
Saving.play = async ({ canvasElement }) => {
  respondWith(() => new Promise(() => {}));
  await typeAndApply(canvasElement);
};

export const Invalid = Template.bind({});
Invalid.args = {};
Invalid.play = async ({ canvasElement }) => {
  respondWith(async (config) => ({ status: 400, statusText: "", data: {}, headers: {}, config }));
  await typeAndApply(canvasElement);
};

export const Applied = Template.bind({});
Applied.args = {};
Applied.play = async ({ canvasElement }) => {
  respondWith(async (config) => ({ status: 200, statusText: "", data: {}, headers: {}, config }));
  await typeAndApply(canvasElement);
};

export const Compact = Template.bind({});
Compact.args = { compact: true };
