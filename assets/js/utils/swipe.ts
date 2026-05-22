interface SwipeOptions {
  onSwipeLeft?: () => void;
  onSwipeRight?: () => void;
  ignoreSelector?: string;
  minDistance?: number;
  maxDuration?: number;
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
    const target = e.target as Element | null;
    ignored = !!(options.ignoreSelector && target?.closest(options.ignoreSelector));
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

  el.addEventListener("touchstart", onTouchStart, { passive: true });
  el.addEventListener("touchend", onTouchEnd, { passive: true });

  return () => {
    el.removeEventListener("touchstart", onTouchStart);
    el.removeEventListener("touchend", onTouchEnd);
  };
}
