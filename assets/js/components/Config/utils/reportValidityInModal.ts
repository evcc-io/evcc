/**
 * `form.reportValidity()` with the document scroll restored afterwards.
 *
 * The native call scroll-into-views the first invalid field and leaks scroll
 * up to `<html>` (overflow:hidden only blocks user scroll). On iOS Safari
 * that wedges the modal's touch scrolling onto the page underneath.
 */
export function reportValidityInModal(form: HTMLFormElement): boolean {
  const { scrollX, scrollY } = window;
  const valid = form.reportValidity();
  if (window.scrollX !== scrollX || window.scrollY !== scrollY) {
    // Chromium ignores scrollTop writes while overflow-y:hidden; relax briefly.
    const html = document.documentElement;
    const prev = html.style.overflowY;
    html.style.overflowY = "auto";
    window.scrollTo(scrollX, scrollY);
    html.style.overflowY = prev;
  }
  return valid;
}
