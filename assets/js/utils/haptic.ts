// the native app shims navigator.vibrate; duration maps to impact strength (<=50 light, <=100 medium)
export function hapticFeedback(style: "light" | "medium" = "light"): void {
  navigator.vibrate?.(style === "medium" ? 60 : 5);
}
