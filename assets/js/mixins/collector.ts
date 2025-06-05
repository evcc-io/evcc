import { defineComponent } from "vue";
import type { State } from "@/types/evcc";

export default defineComponent({
	methods: {
		// collect all target component properties from current instance
		collectProps(component: any, state?: State) {
			let data: Record<string, any> = {};
			for (const prop in component.props) {
				const p = prop as keyof State;
				// check in optional state
				if (state && p in state) {
					data[p] = state[p];
				}
				// check in current instance
				if (p in this) {
					data[p] = (this as Partial<State>)[p];
				}
			}
			return data;
		},
	},
});
