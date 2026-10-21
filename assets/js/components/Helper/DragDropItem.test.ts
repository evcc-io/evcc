import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vite-plus/test";
import DragDropItem from "./DragDropItem.vue";

describe("DragDropItem", () => {
  it("activates only the row, preserving interactive descendants", async () => {
    const wrapper = mount(DragDropItem, {
      props: { interactive: true, selected: true },
      slots: { actions: '<input type="checkbox" />' },
      global: { mocks: { $t: (key: string) => key } },
    });

    await wrapper.trigger("keydown", { key: "Enter" });
    await wrapper.trigger("keydown", { key: " " });
    expect(wrapper.emitted("activate")).toHaveLength(2);
    expect(wrapper.attributes("aria-selected")).toBe("true");

    await wrapper.get("input").trigger("keydown", { key: " " });
    expect(wrapper.emitted("activate")).toHaveLength(2);
    await wrapper.setProps({ disabled: true });

    await wrapper.trigger("click");
    expect(wrapper.emitted("activate")).toHaveLength(2);
  });
});
