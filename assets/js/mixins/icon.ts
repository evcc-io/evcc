import { SIZE } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	props: {
		size: {
			type: String as PropType<SIZE>,
			validator(value: string) {
				return Object.keys(SIZE).includes(value);
			},
			default: SIZE.s,
		},
	},
	computed: {
		svgStyle() {
			const sizes: Record<SIZE, string> = {
				xs: "16px",
				s: "24px",
				m: "32px",
				l: "48px",
				xl: "64px",
			};
			const size = sizes[this.size];
			return { display: "block", width: size, height: size };
		},
	},
});
