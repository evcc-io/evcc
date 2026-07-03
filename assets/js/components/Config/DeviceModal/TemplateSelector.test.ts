import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import TemplateSelector, { type TemplateGroup } from "./TemplateSelector.vue";

config.global.mocks["$t"] = (key: string) => key;

// mirrors the vestel template: multiple products sharing one template
const groups: TemplateGroup[] = [
  {
    label: "generic",
    options: [
      { name: "Ampure Unite", template: "vestel" },
      { name: "Vestel EVC04 Home Smart", template: "vestel" },
      { name: "Webasto Unite", template: "vestel" },
      { name: "Other Box", template: "other" },
    ],
  },
];

function mountSelector(modelValue: string | null = null) {
  return mount(TemplateSelector, {
    props: { deviceType: "charger", isNew: true, modelValue, groups },
  });
}

describe("TemplateSelector", () => {
  test("keeps the chosen product when multiple products share a template", async () => {
    const wrapper = mountSelector();

    await wrapper.find("select").setValue("vestel\tVestel EVC04 Home Smart");
    expect(wrapper.emitted("update:modelValue")).toEqual([["vestel"]]);

    // parent writes the template back via v-model; selection must not snap to "Ampure Unite"
    await wrapper.setProps({ modelValue: "vestel" });
    const select = wrapper.find("select").element as HTMLSelectElement;
    expect(select.selectedOptions[0]?.text.trim()).toBe("Vestel EVC04 Home Smart");
    expect((wrapper.vm as any).getProductName()).toBe("Vestel EVC04 Home Smart");
  });

  test("selects the first matching product when template is set externally", () => {
    const wrapper = mountSelector("vestel");
    expect((wrapper.vm as any).getProductName()).toBe("Ampure Unite");
  });

  test("returns empty product name without selection", () => {
    const wrapper = mountSelector();
    expect((wrapper.vm as any).getProductName()).toBe("");
  });
});
