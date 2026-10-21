import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vite-plus/test";
import CircuitsModal from "./CircuitsModal.vue";
import { ConfigType } from "@/types/evcc";
const circuits = [
  { id: 8, name: "db:8", deviceTitle: "Home", type: ConfigType.Template, config: { parent: "" } },
  {
    id: 9,
    name: "db:9",
    deviceTitle: "Garage",
    type: ConfigType.Template,
    config: { parent: "db:8" },
  },
].map((c) => ({ ...c, deviceProduct: "Static circuit" }));
const setup = () =>
  mount(CircuitsModal, {
    props: {
      circuits,
      loadpoints: [
        { id: 13, title: "Wallbox", circuit: null, editable: true, heating: false },
        { id: 14, title: "Heater", circuit: "db:8", editable: true, heating: true },
      ],
    },
    global: {
      mocks: { $t: (key: string) => key },
      stubs: { GenericModal: { template: "<div><slot /></div>" } },
    },
  });
describe("circuit assignment", () => {
  it("does not highlight YAML loadpoints without an active device", async () => {
    const wrapper = setup();
    await wrapper.setProps({
      loadpoints: [
        {
          name: "yaml-assigned",
          title: "YAML assigned",
          circuit: "db:8",
          editable: false,
          heating: false,
        },
        { name: "yaml-pool", title: "YAML pool", circuit: null, editable: false, heating: false },
        { id: 13, title: "DB pool", circuit: null, editable: true, heating: false },
      ],
    });
    expect(
      wrapper.findAll(
        ".drag-drop-item--selected, .assignment-dragged, .assignment-flash, .assignment-target"
      )
    ).toHaveLength(0);
    expect(wrapper.findAll('[aria-selected="true"]')).toHaveLength(0);
    expect(
      wrapper
        .get('[data-testid="assignment-pool"]')
        .findAll('[role="listitem"]')
        .map((row) => row.text())
    ).toEqual(["DB pool", "YAML pool"]);
    wrapper.unmount();
  });
  it("uses one continuous branch guide through nested loadpoints", async () => {
    const wrapper = setup();
    await wrapper.setProps({
      loadpoints: [
        { id: 14, title: "Heater", circuit: "db:8", editable: true, heating: true },
        { id: 15, title: "Wallbox", circuit: "db:9", editable: true, heating: false },
      ],
    });
    expect(wrapper.findAll(".circuit-branch")).toHaveLength(2);
    expect(wrapper.findAll(".tree-col")).toHaveLength(0);
    expect(wrapper.get('[data-loadpoint="15"]').classes()).toContain("assigned-row");
    expect(wrapper.get('[data-loadpoint="15"]').attributes("style") ?? "").not.toContain("44px");
    wrapper.unmount();
  });
  it("selects a device first, then assigns, and keeps assigned rows before children", async () => {
    const wrapper = setup();
    const row = wrapper.get('[data-loadpoint="13"]');
    expect(wrapper.get('[data-testid="assignment-pool"] h6').classes()).toContain("mt-0");
    expect(row.classes()).toContain("mb-2");
    expect(row.classes()).not.toContain("mb-0");
    await row.trigger("click");
    expect(wrapper.emitted("assign")).toBeUndefined();
    await wrapper.get('[data-circuit="db:9"] button').trigger("click");
    expect(wrapper.emitted("assign")).toEqual([[{ id: 13, circuit: "db:9" }]]);
    expect(wrapper.html().indexOf("Heater")).toBeLessThan(wrapper.html().indexOf("Garage"));
  });
  it("selection of an assigned device does not remove it, pool click does", async () => {
    const wrapper = setup();
    await wrapper.get('[data-loadpoint="14"]').trigger("click");
    expect(wrapper.emitted("assign")).toBeUndefined();
    await wrapper.get('[data-testid="assignment-pool"]').trigger("click");
    expect(wrapper.emitted("assign")).toEqual([[{ id: 14, circuit: null }]]);
  });
  it("blocks pending actions, retains selection on error, clears and flashes only success", async () => {
    vi.useFakeTimers();
    const wrapper = setup();
    await wrapper.get('[data-loadpoint="13"]').trigger("click");
    await wrapper.setProps({ pending: true });
    await wrapper.get('[data-circuit="db:9"] button').trigger("click");
    expect(wrapper.emitted("assign")).toBeUndefined();
    await wrapper.setProps({ pending: false, error: "Rejected" });
    expect(wrapper.get('[data-loadpoint="13"]').attributes("aria-selected")).toBe("true");
    expect(wrapper.get('[role="alert"]').text()).toBe("Rejected");
    await wrapper.setProps({ revision: 1, flashId: 14 });
    expect(wrapper.get('[data-loadpoint="13"]').attributes("aria-selected")).toBe("false");
    expect(wrapper.get('[data-loadpoint="14"]').classes()).toContain("assignment-flash");
    await vi.advanceTimersByTimeAsync(1400);
    expect(wrapper.get('[data-loadpoint="14"]').classes()).not.toContain("assignment-flash");
    wrapper.unmount();
    vi.useRealTimers();
  });
});
