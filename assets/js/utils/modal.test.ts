import { describe, expect, test } from "vite-plus/test";
import Modal from "bootstrap/js/dist/modal";
import { hideModal } from "./modal";

describe("hideModal", () => {
  test("hide requested during show transition is applied once shown", async () => {
    const el = document.createElement("div");
    el.className = "modal fade";
    el.innerHTML = '<div class="modal-dialog"><div class="modal-body"></div></div>';
    document.body.appendChild(el);
    const shown = new Promise((resolve) => el.addEventListener("shown.bs.modal", resolve));
    const hidden = new Promise((resolve) => el.addEventListener("hidden.bs.modal", resolve));

    Modal.getOrCreateInstance(el).show();
    hideModal(el);
    await shown;
    await hidden;

    expect(el.classList.contains("show")).toBe(false);
    expect(el.style.display).toBe("none");
  });
});
