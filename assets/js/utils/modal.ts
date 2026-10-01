import Modal from "bootstrap/js/dist/modal";

// true while bootstrap runs the show fade, during which it ignores hide()
export function isShowing(el: HTMLElement): boolean {
  const instance = Modal.getInstance(el);
  // @ts-expect-error bs internal
  return !!instance?._isShown && !!instance._isTransitioning;
}

export function hideModal(el: HTMLElement): void {
  if (isShowing(el)) {
    el.addEventListener("shown.bs.modal", () => hideModal(el), { once: true });
    return;
  }
  Modal.getOrCreateInstance(el).hide();
}
