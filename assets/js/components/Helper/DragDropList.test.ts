import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vite-plus/test";
import DragDropList from "./DragDropList.vue";
import { dragAndDrop, tearDown, updateConfig } from "@formkit/drag-and-drop";
const { registrations } = vi.hoisted(() => ({ registrations: new WeakMap() }));
vi.mock("@formkit/drag-and-drop", () => ({
  parents: registrations,
  dragAndDrop: vi.fn((registration) => {
    registrations.set(registration.parent, { config: registration.config });
  }),
  tearDown: vi.fn(),
  updateConfig: vi.fn(),
  handleParentDrop: vi.fn(),
  handleNodeDrop: vi.fn(),
  handleNodePointerdown: vi.fn(),
}));
const setup = (props = {}) => {
  const wrapper = mount(DragDropList, {
    props: { values: [13], ...props },
    global: { mocks: { $t: (key: string) => key } },
  });
  return { wrapper, registration: vi.mocked(dragAndDrop).mock.calls.at(-1)![0] as any };
};
describe("DragDropList", () => {
  it("reads replacement values without exposing parent arrays and cleans up", async () => {
    const { wrapper, registration } = setup();
    await wrapper.setProps({ values: [14] });
    expect(registration.getValues()).toEqual([14]);
    expect(registration.getValues()).not.toBe(wrapper.props("values"));
    registration.setValues([14, 13]);
    expect(wrapper.emitted("reorder")).toEqual([[[14, 13]]]);
    const element = wrapper.element;
    wrapper.unmount();
    expect(tearDown).toHaveBeenCalledWith(element);
  });
  it("persists only an actual drop, not transfer hover or cancelled drag", () => {
    const { wrapper, registration } = setup({ transfer: true, name: "db:9", group: "circuits" });
    const config = registration.config;
    const state = {
      initialParent: { data: { config: { name: "pool" } } },
      draggedNode: { data: { value: 13 } },
    };
    config.performTransfer({ targetParent: { data: { config: { name: "db:9" } } } });
    expect(wrapper.emitted("transfer")).toBeUndefined();
    config.onDragend();
    expect(wrapper.emitted("transfer")).toBeUndefined();
    config.handleParentDrop({}, state);
    expect(wrapper.emitted("transfer")).toEqual([[{ value: 13, from: "pool", to: "db:9" }]]);
    config.handleParentDrop(
      {},
      { ...state, initialParent: { data: { config: { name: "db:9" } } } }
    );
    expect(wrapper.emitted("transfer")).toHaveLength(1);
  });
  it("retains transfer configuration when pending changes", async () => {
    const { wrapper } = setup({ transfer: true, name: "db:9", group: "circuits" });
    await wrapper.setProps({ disabled: true });
    expect(updateConfig).toHaveBeenLastCalledWith(
      wrapper.element,
      expect.objectContaining({
        disabled: true,
        name: "db:9",
        group: "circuits",
        performTransfer: expect.any(Function),
      })
    );
  });
});
