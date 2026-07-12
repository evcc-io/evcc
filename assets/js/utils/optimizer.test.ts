import { describe, expect, test } from "vitest";
import { optimizerActionClass } from "./optimizer";

describe("optimizerActionClass", () => {
  test("maps actions to semantic colors", () => {
    expect(optimizerActionClass("charge")).toBe("text-success");
    expect(optimizerActionClass("stop")).toBe("text-danger");
    expect(optimizerActionClass("hold")).toBe("text-warning");
    expect(optimizerActionClass("holdcharge")).toBe("text-warning");
    expect(optimizerActionClass("normal")).toBe("");
    expect(optimizerActionClass(null)).toBe("");
    expect(optimizerActionClass()).toBe("");
  });
});
