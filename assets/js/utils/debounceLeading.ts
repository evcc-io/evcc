import type { Timeout } from "@/types/evcc";

export function debounceLeading<T extends (...args: any[]) => any>(fn: T, delay: number): T {
	let timer: Timeout;
	return ((...args: any[]) => {
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
	}) as T;
}
