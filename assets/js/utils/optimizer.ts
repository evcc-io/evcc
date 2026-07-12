export type OptimizerAction = "normal" | "hold" | "charge" | "holdcharge" | "stop";

// Maps an optimizer suggestion action to a semantic text-color class.
// green = start charging, red = stop, yellow = hold, none = normal operation.
export function optimizerActionClass(action?: OptimizerAction | null): string {
  switch (action) {
    case "charge":
      return "text-success";
    case "stop":
      return "text-danger";
    case "hold":
    case "holdcharge":
      return "text-warning";
    default:
      return "";
  }
}
