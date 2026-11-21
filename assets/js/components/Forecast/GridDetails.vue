<template>
	<div v-if="lines.length" class="row gx-2 mt-1">
		<div
			v-for="line in lines"
			:key="line.key"
			:class="[lines.length > 1 ? 'col-6' : 'col-12', line.align]"
		>
			<button
				type="button"
				class="price-line border-0 bg-transparent p-0 small text-reset"
				:class="[line.align, { 'price-line--dim': !line.shown }]"
				:aria-pressed="line.focused"
				@click="$emit('focus-line', line.key)"
			>
				<span class="d-block text-gray">
					<DeviceColorDot line :color="line.color" class="me-2 align-middle" />
					<span class="price-label">{{ line.label }}</span>
				</span>
				<span
					class="d-flex flex-column flex-md-row column-gap-3 text-gray fw-bold"
					:class="line.justify"
				>
					<span class="text-nowrap">{{ line.summary.avg }}</span>
					<span v-if="line.summary.range" class="text-nowrap">{{
						line.summary.range
					}}</span>
				</span>
			</button>
		</div>
	</div>
	<div v-else-if="gridSummary" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<span class="text-gray">{{ $t("forecast.price.range") }}</span>
				<br />
				<span class="text-price fw-bold">{{ gridSummary.range || gridSummary.avg }}</span>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<span class="text-gray">{{ $t("forecast.price.average") }}</span>
				<br />
				<span class="text-price fw-bold">{{ gridSummary.avg }}</span>
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { CURRENCY, UiForecastSlot } from "@/types/evcc";
import { isStaticTariff } from "@/utils/forecast";
import colors from "@/colors";
import DeviceColorDot from "../Helper/DeviceColorDot.vue";
import { filterForecastSlots } from "./echarts";

export default defineComponent({
	name: "GridDetails",
	components: { DeviceColorDot },
	mixins: [formatter],
	props: {
		grid: { type: Array as PropType<UiForecastSlot[]> },
		feedin: { type: Array as PropType<UiForecastSlot[]> },
		currency: { type: String as PropType<CURRENCY> },
		showGrid: { type: Boolean, default: true },
		showFeedin: { type: Boolean, default: true },
		endDate: { type: Date, required: true },
	},
	emits: ["focus-line"],
	computed: {
		gridSummary(): { avg: string; range: string } | null {
			return this.summarize(this.grid);
		},
		feedinSummary(): { avg: string; range: string } | null {
			return this.summarize(this.feedin);
		},
		lines() {
			const all = [
				{
					key: "grid",
					label: this.$t("main.energyflow.gridImport"),
					color: colors.price || "",
					summary: this.gridSummary,
					shown: this.showGrid,
					focused: !this.showFeedin,
					align: "text-start",
					justify: "",
				},
				{
					key: "feedin",
					label: this.$t("main.energyflow.pvExport"),
					color: colors.export || "",
					summary: this.feedinSummary,
					shown: this.showFeedin,
					focused: !this.showGrid,
					align: "text-end",
					justify: "justify-content-end",
				},
			];
			const available = all.flatMap((l) => (l.summary ? [{ ...l, summary: l.summary }] : []));
			// a flat single tariff has nothing to zoom into
			const flat = available.length === 1 && !available[0]!.summary.range;
			return flat ? [] : available;
		},
	},
	methods: {
		summarize(slots?: UiForecastSlot[]): { avg: string; range: string } | null {
			const upcoming = this.upcomingSlots(slots);
			if (upcoming.length === 0) return null;
			const values = upcoming.map((s) => s.value);
			const avg = values.reduce((a, b) => a + b, 0) / values.length;
			const fmtAvg = this.fmtPricePerKWh(avg, this.currency, false, true);
			if (isStaticTariff(upcoming)) return { avg: fmtAvg, range: "" };
			const range = this.fmtPriceRange(
				Math.min(...values),
				Math.max(...values),
				this.currency
			);
			return { avg: `⌀ ${fmtAvg}`, range };
		},
		upcomingSlots(slots?: UiForecastSlot[]): UiForecastSlot[] {
			return filterForecastSlots(slots || [], new Date(), this.endDate);
		},
	},
});
</script>

<style scoped>
.price-line {
	transition: opacity 150ms;
}
.price-line--dim {
	opacity: 0.35;
}
.price-line:hover .price-label {
	text-decoration: underline;
}
</style>
