export function hapticFeedback(): void {
  navigator.vibrate?.(5);
}
