import type { Timeout } from "@/types/evcc";

export function debounce<T extends (...args: any[]) => any>(fn: T, delay: number): T {
  let timer: Timeout;
  return ((...args: any[]) => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  }) as T;
}
