<template>
	<router-link to="/optimize" class="root" @click.stop>
		{{ label }}: <span class="message">{{ message }}</span>
	</router-link>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { BatteryForecastPoint } from "@/types/evcc";

export default defineComponent({
	name: "ForecastMessage",
	mixins: [formatter],
	props: {
		point: { type: Object as PropType<BatteryForecastPoint>, required: true },
		high: { type: Boolean, default: false },
	},
	computed: {
		label(): string {
			if (this.point.limit) {
				return this.$t(
					this.high
						? "main.energyflow.batteryForecastFull"
						: "main.energyflow.batteryForecastEmpty"
				);
			}
			const soc = this.fmtPercentage(this.point.soc);
			return this.$t(
				this.high
					? "main.energyflow.batteryForecastNextHigh"
					: "main.energyflow.batteryForecastNextLow",
				{ soc }
			);
		},
		message(): string {
			return this.fmtAbsoluteDate(new Date(this.point.time));
		},
	},
});
</script>

<style scoped>
.root {
	color: inherit;
	text-decoration: none;
}
.message {
	text-decoration: underline;
}
</style>
