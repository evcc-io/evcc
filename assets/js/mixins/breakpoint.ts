export type Breakpoint = "xs" | "sm" | "md" | "lg" | "xl" | "xxl";

// bootstrap breakpoints, value = last width still inside
export const MAX_WIDTH: Record<Breakpoint, number> = {
  xs: 575,
  sm: 767,
  md: 991,
  lg: 1199,
  xl: 1399,
  xxl: Infinity,
};

const BREAKPOINTS = Object.entries(MAX_WIDTH) as [Breakpoint, number][];

export function isMobileWidth(width: number = window.innerWidth): boolean {
  return width <= MAX_WIDTH.sm;
}

export default {
  data() {
    return {
      breakpoint: "md" as Breakpoint,
    };
  },
  methods: {
    updateBreakpoint(): void {
      const width: number = window.innerWidth;
      const self = this as any;
      for (const [name, maxWidth] of BREAKPOINTS) {
        if (width <= maxWidth) {
          self.breakpoint = name;
          return;
        }
      }
    },
  },
  mounted() {
    const self = this as any;
    self.updateBreakpoint();
    window.addEventListener("resize", self.updateBreakpoint);
  },
  beforeUnmount() {
    const self = this as any;
    window.removeEventListener("resize", self.updateBreakpoint);
  },
};
