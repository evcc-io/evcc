export type Breakpoint = "xs" | "sm" | "md" | "lg" | "xl" | "xxl";

interface BreakpointDef {
	name: Breakpoint;
	maxWidth: number;
}

const BREAKPOINTS: BreakpointDef[] = [
	{ name: "xs", maxWidth: 575 },
	{ name: "sm", maxWidth: 767 },
	{ name: "md", maxWidth: 991 },
	{ name: "lg", maxWidth: 1199 },
	{ name: "xl", maxWidth: 1399 },
	{ name: "xxl", maxWidth: Infinity },
];

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
			for (const bp of BREAKPOINTS) {
				if (width <= bp.maxWidth) {
					self.breakpoint = bp.name;
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
	beforeDestroy() {
		const self = this as any;
		window.removeEventListener("resize", self.updateBreakpoint);
	},
};
