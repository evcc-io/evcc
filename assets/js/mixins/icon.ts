import { ICON_SIZE } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	props: {
		size: {
			type: String as PropType<ICON_SIZE>,
			validator(value: ICON_SIZE) {
				return Object.values(ICON_SIZE).includes(value);
			},
			default: ICON_SIZE.S,
		},
	},
	computed: {
		svgStyle() {
			const sizes: Record<ICON_SIZE, string> = {
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
