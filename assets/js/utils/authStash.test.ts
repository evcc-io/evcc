import { beforeEach, describe, expect, it } from "vite-plus/test";
import { popAuthValues, stashAuthValues } from "./authStash";

describe("authStash", () => {
  beforeEach(() => window.localStorage.clear());

  it("restores once for the same device", () => {
    stashAuthValues("vehicle:new", "tesla-fleet", { clientid: "id" });
    expect(popAuthValues("vehicle:new")).toMatchObject({
      templateName: "tesla-fleet",
      values: { clientid: "id" },
    });
    expect(popAuthValues("vehicle:new")).toBeNull();
  });

  it("keeps the stash for a different device", () => {
    stashAuthValues("vehicle:new", "tesla-fleet", {});
    expect(popAuthValues("meter:3")).toBeNull();
    expect(popAuthValues("vehicle:new")).not.toBeNull();
  });

  it("drops an expired stash", () => {
    stashAuthValues("vehicle:new", "tesla-fleet", {});
    expect(popAuthValues("vehicle:new", Date.now() + 11 * 60 * 1000)).toBeNull();
    expect(popAuthValues("vehicle:new")).toBeNull();
  });
});
