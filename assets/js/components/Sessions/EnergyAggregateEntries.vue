<template>
	<div class="root">
		<div
			v-for="entry in entries"
			:key="entry.name"
			:style="{ borderLeft: `8px solid ${entry.color}` }"
			class="ps-3"
		>
			<strong :style="{ color: entry.color }" class="fs-6">{{ entry.name }}</strong>
			<div class="fs-6">{{ fmtPercentage(entry.percent) }}</div>
			<div class="fs-6">{{ fmtKWh(entry.energy * 1e3) }}</div>
		</div>
	</div>
</template>

<script>
import formatter from "../../mixins/formatter";

export default {
	name: "EnergyAggregateEntries",
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: "loadpoint" },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
	},
	mixins: [formatter],
	computed: {
		entries() {
			const aggregatedData = {};
			let totalEnergy = 0;

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = 0;
				}
				aggregatedData[groupKey] += session.chargedEnergy;
				totalEnergy += session.chargedEnergy;
			});

			const entries = Object.entries(aggregatedData)
				.map(([key, value]) => ({
					name: key,
					energy: value,
					percent: (100 / totalEnergy) * value,
					color: this.colorMappings[this.groupBy][key],
				}))
				.sort((a, b) => b.energy - a.energy);

			return entries;
		},
	},
};
</script>

<style scoped>
.root {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 2rem;
}
</style>
