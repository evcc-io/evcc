interface SwipeOptions {
  onSwipeLeft?: () => void;
  onSwipeRight?: () => void;
  minDistance?: number;
  maxDuration?: number;
}

const DWELL_MS = 100;
const DWELL_SLOP_PX = 10;

// Touch tooltip gate: the chart never sees raw touch events (zrender would otherwise
// ignore mouse events for 700ms after every touchend). The tooltip appears only after
// the finger rests for a moment, then follows it; fast movement (swipes, scrolling)
// keeps it hidden and a pause re-arms it. The tooltip is driven by a synthetic
// mousemove at the finger position, which both chart libs handle natively.
export function attachTouchTooltipGate(el: HTMLElement, hide: () => void): () => void {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let shown = false;
  let latched = false;
  let x = 0;
  let y = 0;
  let anchorX = 0;
  let anchorY = 0;
  let gestureX = 0;
  let gestureY = 0;

  const show = () => {
    const target = document.elementFromPoint(x, y);
    if (!target || !el.contains(target)) return;
    shown = true;
    const event = new MouseEvent("mousemove", { clientX: x, clientY: y, bubbles: true });
    // WebKit computes offsetX/Y of synthetic events against the wrong base element,
    // and both chart libs read them; provide correct values relative to the chart
    const rect = el.getBoundingClientRect();
    Object.defineProperty(event, "offsetX", { value: x - rect.left });
    Object.defineProperty(event, "offsetY", { value: y - rect.top });
    target.dispatchEvent(event);
  };

  const restartDwell = () => {
    if (timer) clearTimeout(timer);
    anchorX = x;
    anchorY = y;
    timer = setTimeout(show, DWELL_MS);
  };

  const onTouchStart = (e: TouchEvent) => {
    const touch = e.touches[0];
    if (!touch) return;
    e.stopPropagation();
    // WebKit long-press jumps selection to the nearest selectable content
    // outside the chart; block selection page-wide for the gesture
    document.body.classList.add("user-select-none");
    shown = false;
    latched = false;
    x = touch.clientX;
    y = touch.clientY;
    gestureX = x;
    gestureY = y;
    restartDwell();
  };

  const onTouchMove = (e: TouchEvent) => {
    const touch = e.touches[0];
    if (!touch) return;
    e.stopPropagation();
    x = touch.clientX;
    y = touch.clientY;
    // large displacement before the tooltip ever showed = scroll or swipe,
    // no tooltip for the rest of this gesture (scroll deceleration would
    // otherwise pass the dwell check while the page is still moving)
    if (!shown && !latched && Math.hypot(x - gestureX, y - gestureY) > 30) {
      latched = true;
      if (timer) clearTimeout(timer);
    }
    if (latched) return;
    if (shown) {
      show();
    } else if (Math.hypot(x - anchorX, y - anchorY) > DWELL_SLOP_PX) {
      restartDwell();
    }
  };

  const onTouchEnd = (e: TouchEvent) => {
    e.stopPropagation();
    // suppress compat mouse events after touch, they would re-trigger the tooltip
    if (e.cancelable) e.preventDefault();
    if (timer) clearTimeout(timer);
    shown = false;
    document.body.classList.remove("user-select-none");
    hide();
  };

  el.addEventListener("touchstart", onTouchStart, { capture: true, passive: true });
  el.addEventListener("touchmove", onTouchMove, { capture: true, passive: true });
  el.addEventListener("touchend", onTouchEnd, { capture: true });
  el.addEventListener("touchcancel", onTouchEnd, { capture: true });

  return () => {
    if (timer) clearTimeout(timer);
    document.body.classList.remove("user-select-none");
    el.removeEventListener("touchstart", onTouchStart, { capture: true });
    el.removeEventListener("touchmove", onTouchMove, { capture: true });
    el.removeEventListener("touchend", onTouchEnd, { capture: true });
    el.removeEventListener("touchcancel", onTouchEnd, { capture: true });
  };
}

export function attachSwipeHandler(el: HTMLElement, options: SwipeOptions): () => void {
  const minDistance = options.minDistance ?? 60;
  const maxDuration = options.maxDuration ?? 600;
  let startX = 0;
  let startY = 0;
  let startTime = 0;
  let ignored = false;

  const onTouchStart = (e: TouchEvent) => {
    if (e.touches.length !== 1) {
      ignored = true;
      return;
    }
    ignored = false;
    startX = e.touches[0].clientX;
    startY = e.touches[0].clientY;
    startTime = Date.now();
  };

  const onTouchEnd = (e: TouchEvent) => {
    if (ignored) return;
    const touch = e.changedTouches[0];
    if (!touch) return;
    const dx = touch.clientX - startX;
    const dy = touch.clientY - startY;
    if (Date.now() - startTime > maxDuration) return;
    if (Math.abs(dx) < minDistance) return;
    if (Math.abs(dy) >= Math.abs(dx)) return;
    if (dx < 0) options.onSwipeLeft?.();
    else options.onSwipeRight?.();
  };

  // capture phase, so the touch tooltip gate's stopPropagation on chart elements
  // cannot starve the swipe detection
  el.addEventListener("touchstart", onTouchStart, { capture: true, passive: true });
  el.addEventListener("touchend", onTouchEnd, { capture: true, passive: true });

  return () => {
    el.removeEventListener("touchstart", onTouchStart, { capture: true });
    el.removeEventListener("touchend", onTouchEnd, { capture: true });
  };
}
