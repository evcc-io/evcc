import type { Timeout } from "@/types/evcc";

/**
 * Creates a debounced version of `fn` that calls at the leading edge.
 * The debounced function does not return the result of `fn`, even if `fn` is async.
 */
export function debounceLeading<T extends (...args: any[]) => any>(
  fn: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timer: Timeout;
  return (...args: Parameters<T>) => {
    if (!timer) {
      fn(...args);
      timer = setTimeout(() => {
        timer = null;
      }, delay);
      return;
    }
    clearTimeout(timer);
    timer = setTimeout(() => {
      timer = null;
      fn(...args);
    }, delay);
  };
}
