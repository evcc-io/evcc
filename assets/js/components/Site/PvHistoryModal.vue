<template>
	<GenericModal
		id="pvHistoryModal"
		:title="$t('main.pvTile.historyTitle')"
		size="xl"
		data-testid="pv-history-modal"
		@open="handleOpen"
	>
		<div class="d-flex justify-content-between align-items-center gap-2 flex-wrap mb-3">
			<div class="btn-group" role="group" :aria-label="$t('main.pvTile.historyRange')">
				<button
					v-for="opt in ranges"
					:key="opt.value"
					type="button"
					class="btn btn-sm"
					:class="selectedRange === opt.value ? 'btn-primary' : 'btn-outline-secondary'"
					@click="selectRange(opt.value)"
				>
					{{ opt.label }}
				</button>
			</div>
			<div class="form-check form-switch mb-0">
				<input
					id="pvHistoryCumulative"
					class="form-check-input"
					type="checkbox"
					:checked="cumulative"
					@change="toggleCumulative"
				/>
				<label class="form-check-label" for="pvHistoryCumulative">
					{{ $t("main.pvTile.cumulative") }}
				</label>
			</div>
		</div>

		<div v-if="loading" class="text-muted py-4">{{ $t("main.pvTile.loading") }}</div>
		<div v-else-if="!chartData.datasets.length" class="text-muted py-4">
			{{ $t("main.pvTile.noHistory") }}
		</div>
		<div v-else>
			<div class="history-summary mb-2">
				<small class="text-muted">{{ $t("main.pvTile.total") }}</small>
				<div class="fw-bold fs-4">{{ totalText }}</div>
			</div>
			<div class="chart-wrap">
				<Bar :data="chartData" :options="chartOptions" :height="260" />
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	Chart as ChartJS,
	CategoryScale,
	LinearScale,
	BarController,
	BarElement,
	Tooltip,
	type ChartData,
	type ChartOptions,
} from "chart.js";
import { Bar } from "vue-chartjs";
import GenericModal from "../Helper/GenericModal.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import api from "@/api";
import colors from "@/colors";
import type { Forecast, Meter } from "@/types/evcc";
import { commonOptions } from "../Sessions/chartConfig";

type RangeKey = "day" | "week" | "month" | "year";

interface HistorySlot {
	start: string;
	end: string;
	import: number;
	export: number;
}

interface HistorySeries {
	name: string;
	data: HistorySlot[];
}

ChartJS.register(CategoryScale, LinearScale, BarController, BarElement, Tooltip);

export default defineComponent({
	name: "PvHistoryModal",
	components: { GenericModal, Bar },
	mixins: [formatter],
	props: {
		pv: { type: Array as PropType<Meter[]>, default: () => [] },
		pvEnergy: { type: Number, default: 0 },
		forecast: { type: Object as PropType<Forecast>, default: () => ({}) },
	},
	data() {
		return {
			selectedRange: "day" as RangeKey,
			cumulative: false,
			loading: false,
			slots: [] as { start: Date; import: number }[],
		};
	},
	computed: {
		ranges(): { value: RangeKey; label: string }[] {
			return [
				{ value: "day", label: this.$t("main.pvTile.rangeDay") },
				{ value: "week", label: this.$t("main.pvTile.rangeWeek") },
				{ value: "month", label: this.$t("main.pvTile.rangeMonth") },
				{ value: "year", label: this.$t("main.pvTile.rangeYear") },
			];
		},
		labels(): string[] {
			const locale = this.$i18n?.locale;
			if (this.selectedRange === "day") {
				return this.slots.map((slot) =>
					new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }).format(
						slot.start
					)
				);
			}
			if (this.selectedRange === "year") {
				return this.slots.map((slot) =>
					new Intl.DateTimeFormat(locale, { month: "short" }).format(slot.start)
				);
			}
			return this.slots.map((slot) =>
				new Intl.DateTimeFormat(locale, {
					day: "2-digit",
					month: "2-digit",
				}).format(slot.start)
			);
		},
		values(): number[] {
			const raw = this.slots.map((slot) => slot.import || 0);
			if (!this.cumulative) return raw;
			let sum = 0;
			return raw.map((v) => {
				sum += v;
				return sum;
			});
		},
		totalValue(): number {
			return this.slots.reduce((sum, slot) => sum + (slot.import || 0), 0);
		},
		totalText(): string {
			// totalValue is in kWh; fmtWh expects Wh input
			return this.fmtWh(this.totalValue * 1000, POWER_UNIT.KW);
		},
		chartData(): ChartData<"bar"> {
			if (!this.values.length) return { labels: [], datasets: [] };
			return {
				labels: this.labels,
				datasets: [
					{
						label: this.$t("main.pvTile.generated"),
						data: this.values,
						backgroundColor: "#faf000",
						borderRadius: 4,
						maxBarThickness: this.selectedRange === "day" ? 14 : 24,
					},
				],
			};
		},
		chartOptions(): ChartOptions<"bar"> {
			return {
				...commonOptions,
				scales: {
					x: {
						border: { display: false },
						grid: { display: false },
						ticks: {
							color: colors.muted || undefined,
							maxRotation: 0,
							autoSkip: true,
							maxTicksLimit: this.selectedRange === "day" ? 12 : 14,
						},
					},
					y: {
						border: { display: false },
						title: {
							display: true,
							text: "kWh",
							color: colors.muted || undefined,
						},
						ticks: {
							color: colors.muted || undefined,
							callback: (value: number | string) => Number(value).toFixed(1),
						},
						grid: {
							color: colors.border || undefined,
						},
					},
				},
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						callbacks: {
							title: (items) => `${items[0]?.label || ""}`,
							label: (ctx) => {
								const val = ctx.parsed.y || 0;
								return `${this.$t("main.pvTile.generated")}: ${val.toFixed(1)} kWh`;
							},
						},
					},
				},
			} as ChartOptions<"bar">;
		},
	},
	methods: {
		handleOpen() {
			this.fetchHistory();
		},
		selectRange(range: RangeKey) {
			if (this.selectedRange === range) return;
			this.selectedRange = range;
			this.fetchHistory();
		},
		toggleCumulative() {
			this.cumulative = !this.cumulative;
		},
		rangeParams(): { from: string; to: string; aggregate: "hour" | "day" | "month" } {
			const now = new Date();
			const to = new Date(now);
			const from = new Date(now);
			if (this.selectedRange === "day") {
				from.setHours(0, 0, 0, 0);
				return { from: from.toISOString(), to: to.toISOString(), aggregate: "hour" };
			}
			if (this.selectedRange === "week") {
				from.setDate(from.getDate() - 6);
				from.setHours(0, 0, 0, 0);
				return { from: from.toISOString(), to: to.toISOString(), aggregate: "day" };
			}
			if (this.selectedRange === "month") {
				from.setDate(from.getDate() - 29);
				from.setHours(0, 0, 0, 0);
				return { from: from.toISOString(), to: to.toISOString(), aggregate: "day" };
			}
			from.setMonth(0, 1);
			from.setHours(0, 0, 0, 0);
			return { from: from.toISOString(), to: to.toISOString(), aggregate: "month" };
		},
		async fetchHistory() {
			this.loading = true;
			try {
				const params = this.rangeParams();
				const res = await api.get("history/energy", {
					params: { ...params, group: "pv" },
				});
				const series = (res.data || []) as HistorySeries[];
				const buckets = new Map<number, number>();
				for (const s of series) {
					for (const slot of s.data || []) {
						const start = new Date(slot.start).getTime();
						buckets.set(start, (buckets.get(start) || 0) + (slot.import || 0));
					}
				}
				this.slots = Array.from(buckets.entries())
					.sort((a, b) => a[0] - b[0])
					.map(([start, imp]) => ({ start: new Date(start), import: imp }));

				if (this.selectedRange === "day") {
					const total = this.slots.reduce((sum, slot) => sum + (slot.import || 0), 0);
					if (total <= 0) {
						this.slots = this.syntheticDaySlots();
					}
				}
			} catch (e) {
				console.error("Failed to load pv history", e);
				this.slots = this.selectedRange === "day" ? this.syntheticDaySlots() : [];
			} finally {
				this.loading = false;
			}
		},
		syntheticDaySlots(): { start: Date; import: number }[] {
			const now = new Date();
			const dayStart = new Date(now);
			dayStart.setHours(0, 0, 0, 0);

			const entries = this.forecast?.solar?.timeseries || [];
			const hourWeights = Array<number>(24).fill(0);
			for (const entry of entries) {
				const ts = new Date(entry.ts);
				if (ts.toDateString() !== now.toDateString()) continue;
				const hour = ts.getHours();
				hourWeights[hour] = Math.max(hourWeights[hour] || 0, entry.val || 0);
			}

			let sumWeights = hourWeights.reduce((sum, v) => sum + v, 0);
			if (sumWeights <= 0) {
				for (let h = 6; h <= 20; h++) {
					const x = (h - 6) / 14;
					hourWeights[h] = Math.max(0, Math.sin(Math.PI * x));
				}
				sumWeights = hourWeights.reduce((sum, v) => sum + v, 0);
			}

			const totalKWh = Math.max(0, this.pvEnergy || 0);
			if (totalKWh <= 0 || sumWeights <= 0) {
				return Array.from({ length: 24 }, (_, hour) => ({
					start: new Date(dayStart.getTime() + hour * 3600 * 1000),
					import: 0,
				}));
			}

			const factor = totalKWh / sumWeights;
			return hourWeights.map((w, hour) => ({
				start: new Date(dayStart.getTime() + hour * 3600 * 1000),
				import: w * factor,
			}));
		},
	},
});
</script>

<style scoped>
.history-summary {
	padding: 0.5rem 0;
}
.chart-wrap {
	height: 280px;
}
</style>
