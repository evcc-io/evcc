import OnboardingModal from "./OnboardingModal.vue";
import configModal from "@/configModal";
import auth from "@/components/Auth/auth";
import { clearSession } from "@/onboarding";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Config/OnboardingModal",
  component: OnboardingModal,
} as Meta<typeof OnboardingModal>;

type Args = InstanceType<typeof OnboardingModal>["$props"] & {
  step: string;
  passwordMissing?: boolean;
};

const Template: StoryFn<Args> = (args) => ({
  components: { OnboardingModal },
  setup() {
    return { args };
  },
  template: '<OnboardingModal v-bind="args" />',
  mounted() {
    auth.configured = !args.passwordMissing;
    clearSession();
    configModal.stack = [{ name: "onboarding", type: args.step }];
    const el = document.getElementById("onboardingModal");
    if (el) {
      import("bootstrap/js/dist/modal").then(({ default: Modal }) => {
        Modal.getOrCreateInstance(el).show();
      });
    }
  },
});

export const Password = Template.bind({});
Password.args = { step: "password", passwordMissing: true };

export const AuthDisabled = Template.bind({});
AuthDisabled.args = { step: "password", authDisabled: true };

export const Site = Template.bind({});
Site.args = { step: "site" };

export const Battery = Template.bind({});
Battery.args = { step: "battery" };

export const GridConfigured = Template.bind({});
GridConfigured.args = {
  step: "grid",
  gridMeter: { id: 1, name: "meter1", type: "template", deviceProduct: "Demo meter", config: {} },
  deviceValues: { meter: { meter1: { power: { value: 1200 } } } },
};

export const Control = Template.bind({});
Control.args = { step: "control" };

export const Done = Template.bind({});
Done.args = { step: "done" };
