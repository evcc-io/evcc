<template>
	<Card
		:title="$t('energy.group.meter')"
		edge-to-edge
		class="box-pull-out mb-4"
		data-testid="energy-meters"
		data-anchor
	>
		<EntityDetail
			v-for="(m, i) in meters"
			:key="m.title"
			:class="{ 'mt-3': i > 0 }"
			:title="m.title"
			:color="colors[m.title] || ''"
			:device-color="deviceColors[m.title] || ''"
			pickable
			:series="{ ...m, paletteIndex: i }"
			:period="period"
			:from="from"
			:to="to"
			:chart-toggle="false"
			:chart-height="130"
			:show-x-axis="i === meters.length - 1"
			subhead
			@drill="$emit('drill', $event)"
		>
			<template #aside>
				<div class="col-4 col-lg-12">
					<Stat
						:label="$t('energy.direction.meter.energy')"
						:number="sumEnergy(m)"
						:format="(v: number) => fmtKWh(v)"
						compact
					/>
				</div>
				<div v-if="sumEnergy(m, 'returnEnergy') > 0" class="col-4 col-lg-12">
					<Stat
						:label="$t('energy.direction.meter.returnEnergy')"
						align="center lg-start"
						:number="sumEnergy(m, 'returnEnergy')"
						:format="(v: number) => fmtKWh(v)"
						compact
					/>
				</div>
			</template>
		</EntityDetail>
	</Card>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Card from "../Helper/Card.vue";
import EntityDetail from "./EntityDetail.vue";
import Stat from "./Stat.vue";
import type { HistorySeries } from "./GroupChart.vue";
import { sumEnergy } from "./mix";
import formatter from "@/mixins/formatter";
import type { DeviceColors } from "@/types/evcc";
import { PERIODS } from "../Sessions/types";

// additional meters outside the energy balance: bars per bucket and the metered sums
export default defineComponent({
	name: "MetersCard",
	components: { Card, EntityDetail, Stat },
	mixins: [formatter],
	props: {
		meters: { type: Array as PropType<HistorySeries[]>, default: () => [] },
		colors: { type: Object as PropType<Record<string, string>>, default: () => ({}) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
		period: { type: String as PropType<PERIODS>, required: true },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
	},
	emits: ["drill"],
	methods: { sumEnergy },
});
</script>
