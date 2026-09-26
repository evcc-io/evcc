<template>
	<div>
		<div ref="chartEl" class="charge-chart my-3"></div>
		<LegendList :legends="legends" :device-colors="deviceColors" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	axisNameStyle,
	FONT_FAMILY,
	forecastYAxis,
	lineCasing,
	tooltipStyle,
	tooltipTable,
	type TooltipRow,
	lineDefaults,
} from "../Forecast/echarts";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import type { BatteryDetail, DemandDetail, DeviceColors } from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import echartsChart from "@/mixins/echartsChart";
import colors from "@/colors";
import { energyAxisScale, type EnergyAxisScale } from "@/utils/energyAxis";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import {
	slotTimes,
	slotXAxis,
	dayBoundaryAxis,
	dayBoundarySeries,
	formatSlotRange,
	whToKW,
	loadpointTitle,
	demandTitle,
	transientHoverDot,
} from "./chart";

const GRID_LABEL = "Grid Power";
const SOLAR_LABEL = "Solar Forecast";

// one stacked bar series, data in kW
interface StackEntry {
	label: string;
	color: string;
	data: number[];
	id?: string;
}

export default defineComponent({
	name: "ChargeChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		evopt: {
			type: Object as PropType<EvoptData>,
			required: true,
		},
		batteryDetails: {
			type: Array as PropType<BatteryDetail[]>,
			required: true,
		},
		demandDetails: {
			type: Array as PropType<DemandDetail[]>,
			default: () => [],
		},
		demandColors: {
			type: Array as PropType<string[]>,
			default: () => [],
		},
		timestamp: {
			type: String,
			default: "",
		},
		batteryColors: {
			type: Array as PropType<string[]>,
			default: () => [],
		},
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	computed: {
		consumptionLabel(): string {
			return this.$t("main.history.group.consumer");
		},
		consumptionColor(): string {
			return colors.muted || "";
		},
		// stack order from the zero line: fixed loads first (consumption, unmodelled, heating),
		// flexible ones outer (charging loadpoints, home batteries)
		stackEntries(): StackEntry[] {
			// the summed gt when there is no breakdown
			const details: DemandDetail[] = this.demandDetails.length
				? this.demandDetails
				: [{ type: "home", values: this.evopt.req.time_series.gt }];
			return [
				...details.map((d, i) => ({
					label: demandTitle(d, this.consumptionLabel),
					color: this.demandColors[i] || this.consumptionColor,
					data: d.values.map(this.toKW),
					id: d.title ? loadpointTitle(d) : undefined,
				})),
				...this.batteryEntries("vehicle"),
				...this.batteryEntries("battery"),
			];
		},
		times(): number[] {
			return slotTimes(this.timestamp, this.evopt.req.time_series.dt);
		},
		gridPower(): number[] {
			const gridImport = this.evopt.res.grid_import || [];
			const gridExport = this.evopt.res.grid_export || [];
			return gridImport.map((imp, i) => {
				const importKW = this.toKW(imp, i);
				const exportKW = this.toKW(gridExport[i] || 0, i);
				return importKW > 0 ? importKW : -exportKW;
			});
		},
		// series values are kW, the shared scale works in W
		axisScale(): EnergyAxisScale {
			const peak = Math.max(
				0,
				...this.chartSeries.flatMap((s) => (s["data"] as number[]).map(Math.abs))
			);
			return energyAxisScale(peak * 1000);
		},
		chartSeries(): Record<string, unknown>[] {
			const grid = {
				name: GRID_LABEL,
				type: "line",
				z: 4,
				data: this.gridPower,
				smooth: 0.2,
				...transientHoverDot(colors.grid || ""),
				lineStyle: { color: colors.grid || "", ...lineDefaults },
			};
			const solar = {
				name: SOLAR_LABEL,
				type: "line",
				z: 4,
				data: this.evopt.req.time_series.ft.map(this.toKW),
				smooth: 0.2,
				...transientHoverDot(colors.forecast || ""),
				lineStyle: { color: colors.forecast || "", ...lineDefaults },
			};
			const series: Record<string, unknown>[] = [
				dayBoundarySeries(this.times),
				lineCasing(grid, 3),
				grid,
				lineCasing(solar, 3),
				solar,
				...this.stackEntries.map((e) => ({
					name: e.label,
					type: "bar",
					stack: "charge",
					// one path per series instead of an svg element per slot
					large: true,
					largeThreshold: 0,
					data: e.data,
					itemStyle: { color: e.color },
					emphasis: { disabled: true },
				})),
			];
			return series;
		},
		chartOption(): Record<string, unknown> {
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { top: 28, right: 36, bottom: 34, left: 0, borderWidth: 0 },
				tooltip: {
					trigger: "axis",
					axisPointer: {
						type: "line",
						lineStyle: { color: colors.muted || "", opacity: 0.4 },
					},
					...tooltipStyle(colors.text || ""),
					formatter: this.tooltipFormatter,
				},
				xAxis: [slotXAxis(this.times, this.weekdayShort), dayBoundaryAxis(this.times)],
				yAxis: forecastYAxis({
					min: undefined,
					position: "right",
					// the day boundary value axis contains 0, keep the axis on the right
					axisLine: { show: false, onZero: false },
					splitNumber: 5,
					name: this.axisScale.unit,
					...axisNameStyle(),
					axisLabel: {
						color: colors.muted || "",
						formatter: (v: number) =>
							this.fmtW(v * 1000, this.axisScale.unit, false, this.axisScale.digits),
					},
				}),
				series: this.chartSeries,
			};
		},
		legends(): Legend[] {
			const legends: Legend[] = [
				{ label: GRID_LABEL, color: colors.grid || "", value: "", type: "line" },
				{ label: SOLAR_LABEL, color: colors.forecast || "", value: "", type: "line" },
				...this.stackEntries.map((e) => ({
					label: e.label,
					color: e.color,
					value: "",
					type: "area" as const,
					id: e.id,
				})),
			];
			return legends;
		},
	},
	methods: {
		toKW(wh: number, index: number): number {
			return whToKW(wh, this.evopt.req.time_series.dt[index] || 0);
		},
		formatValue(value: number): string {
			return this.fmtW(value * 1000, POWER_UNIT.AUTO);
		},
		batteryEntries(type: BatteryDetail["type"]): StackEntry[] {
			return this.batteryDetails.flatMap((detail, index) => {
				const battery = this.evopt.res.batteries[index];
				if (detail.type !== type || !battery) return [];
				// charging positive, discharging negative; one of both is always zero
				const data = battery.charging_power.map((charging, i) => {
					const chargingKW = this.toKW(charging, i);
					const dischargingKW = this.toKW(battery.discharging_power[i] || 0, i);
					return chargingKW > 0 ? chargingKW : -dischargingKW;
				});
				return [
					{
						label: detail.title || detail.name,
						color: this.batteryColors[index] || "",
						data,
						id: type === "vehicle" ? loadpointTitle(detail) : undefined,
					},
				];
			});
		},
		tooltipFormatter(
			params: { dataIndex: number; seriesName?: string; value?: number }[]
		): string {
			const arr = Array.isArray(params) ? params : [params];
			if (!arr.length) return "";
			const dt = this.evopt.req.time_series.dt;
			const head = formatSlotRange(this.times, dt, arr[0]!.dataIndex);
			const rows: TooltipRow[] = arr
				.filter((p) => p.value != null && !Number.isNaN(p.value))
				.filter((p) => !p.seriesName?.endsWith("-casing"))
				.map((p) => {
					const value = p.value as number;
					if (p.seriesName === GRID_LABEL) {
						const name = value > 0 ? "Grid Import" : value < 0 ? "Grid Export" : "Grid";
						return { name, values: [this.formatValue(Math.abs(value))] };
					}
					return { name: p.seriesName, values: [this.formatValue(value)] };
				});
			return tooltipTable(head, rows);
		},
	},
});
</script>

<style scoped>
.charge-chart {
	height: 300px;
	width: 100%;
}
</style>
