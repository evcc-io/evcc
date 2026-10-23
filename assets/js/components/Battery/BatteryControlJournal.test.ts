import { mount, flushPromises } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import BatteryControlJournal from "./BatteryControlJournal.vue";
import api from "@/api";

vi.mock("@/api", () => ({ default: { post: vi.fn() } }));

function card(state = "control-conflict") {
  return mount(BatteryControlJournal, {
    props: {
      controls: {
        "db:3": {
          state,
          reason: "unknown manual",
          actualMode: "manual",
          nativeMode: "1",
          revision: "r1",
          expectedPower: 400,
        },
      },
    },
    global: { mocks: { $t: (key: string) => key } },
  });
}

describe("battery control journal decisions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  it("never submits a decision on opening or shows active states as conflicts", () => {
    const wrapper = card("active");
    expect(wrapper.find('[role="alert"]').exists()).toBe(false);
    expect(api.post).not.toHaveBeenCalled();
    wrapper.unmount();
  });
  it("sends explicit recovery with the displayed revision", async () => {
    vi.mocked(api.post).mockResolvedValue({} as never);
    const wrapper = card();
    expect(api.post).not.toHaveBeenCalled();
    await wrapper.get('[data-testid="pv-recover"]').trigger("click");
    await flushPromises();
    expect(api.post).toHaveBeenCalledWith("batterypvcontrol", {
      battery: "db:3",
      action: "recover",
      revision: "r1",
    });
    wrapper.unmount();
  });
  it("keeps the conflict visible when persistence fails", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("disk full"));
    const wrapper = card();
    await wrapper.get('[data-testid="pv-disable"]').trigger("click");
    await flushPromises();
    expect(wrapper.find('[role="alert"]').exists()).toBe(true);
    expect(wrapper.text()).toContain("battery.control.failed");
    wrapper.unmount();
  });
  it("keeps disabled control visible and permits only explicit recovery", () => {
    const wrapper = card("disabled");
    expect(wrapper.find('[data-testid="pv-disable"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="pv-recover"]').exists()).toBe(true);
    expect(api.post).not.toHaveBeenCalled();
    wrapper.unmount();
  });
});

it("shows pending confirmation without offering a conflict decision", () => {
  const wrapper = card("confirming");
  expect(wrapper.text()).toContain("battery.control.confirming");
  expect(wrapper.find('[data-testid="pv-recover"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="pv-disable"]').exists()).toBe(false);
  wrapper.unmount();
});
it("marks invalid observations instead of presenting old values as current", async () => {
  const wrapper = card();
  await wrapper.setProps({
    controls: {
      "db:3": {
        state: "control-conflict",
        reason: "offline",
        actualMode: "manual",
        nativeMode: "1",
        observationValid: false,
        observedAt: "2026-10-11T07:44:11Z",
      },
    },
  });
  expect(wrapper.text()).toContain("battery.control.stale");
  expect(wrapper.text()).toContain("2026-10-11T07:44:11Z");
  wrapper.unmount();
});
