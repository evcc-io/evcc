import { describe, expect, test, vi } from "vite-plus/test";
import { normalizeFeatureName, fetchActiveFeatureNames } from "./activeFeatures";
import api from "@/api";

vi.mock("@/api", () => ({
  default: { get: vi.fn() },
  allowClientError: { validateStatus: (status: number) => status >= 200 && status < 500 },
}));

describe("normalizeFeatureName", () => {
  test("normalizes to a comparable key", () => {
    expect(normalizeFeatureName("Home Assistant")).toBe("homeassistant");
    expect(normalizeFeatureName("homeassistant")).toBe("homeassistant");
    expect(normalizeFeatureName("OpenDTU")).toBe("opendtu");
  });
});

describe("fetchActiveFeatureNames", () => {
  test("collects deviceProduct and type from all device classes", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "config/devices/charger") {
        return { status: 200, data: [{ type: "template", deviceProduct: "Home Assistant" }] };
      }
      if (url === "config/devices/meter") {
        return { status: 200, data: [{ type: "opendtu", deviceProduct: "OpenDTU" }] };
      }
      return { status: 200, data: [] };
    });

    const names = await fetchActiveFeatureNames();
    expect(names).toContain("Home Assistant");
    expect(names).toContain("template");
    expect(names).toContain("OpenDTU");
    expect(names).toContain("opendtu");
  });

  test("resolves to an empty list when unauthenticated", async () => {
    vi.mocked(api.get).mockResolvedValue({ status: 401, data: null });
    const names = await fetchActiveFeatureNames();
    expect(names).toEqual([]);
  });

  test("resolves to an empty list on request errors", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("network error"));
    const names = await fetchActiveFeatureNames();
    expect(names).toEqual([]);
  });
});
