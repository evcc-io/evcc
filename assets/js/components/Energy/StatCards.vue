<template>
	<div class="d-flex flex-wrap flex-lg-column gap-3 gap-lg-4 h-100">
		<Card
			v-for="stat in stats"
			:key="stat.key"
			class="stat-card d-flex flex-column justify-content-lg-center"
			:data-testid="`energy-stat-${stat.key}`"
		>
			<Stat
				:label="stat.label"
				:number="stat.number"
				:format="stat.format"
				:sub="stat.sub"
				:tooltip="stat.tooltip"
				:value-class="stat.accent"
			>
				<!-- every missing value is missing for the lack of a tariff -->
				<template v-if="stat.empty">
					<div class="text-muted small">{{ stat.empty }}</div>
					<router-link to="/config#tariffs" class="small">
						{{ $t("energy.stat.configure") }}
					</router-link>
				</template>
				<slot :name="stat.key" />
			</Stat>
		</Card>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Stat from "./Stat.vue";
import Card from "../Helper/Card.vue";
import type { StatItem } from "./types";

// beside a chart on large screens the cards share its height with centered content,
// below that they sit side by side with top aligned content
export default defineComponent({
	name: "StatCards",
	components: { Stat, Card },
	props: {
		stats: { type: Array as PropType<StatItem[]>, default: () => [] },
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

/* equal widths when wrapping into rows, equal heights when stacked */
.stat-card {
	flex: 1 1 10rem;
}
@media (--lg-and-up) {
	.stat-card {
		flex: 1 1 0;
		/* otherwise the content height becomes the base size and shares stay unequal */
		min-height: 0;
	}
}
</style>
