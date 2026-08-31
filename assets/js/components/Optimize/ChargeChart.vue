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
	tooltipStyle,
	tooltipTable,
	type TooltipRow,
} from "../Forecast/echarts";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import type { BatteryDetail, DeviceColors } from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import echartsChart from "@/mixins/echartsChart";
import colors from "@/colors";
import { energyAxisScale, type EnergyAxisScale } from "@/utils/energyAxis";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import { slotTimes, slotXAxis, formatSlotRange, whToKW, loadpointTitle } from "./chart";

const GRID_LABEL = "Grid Power";
const SOLAR_LABEL = "Solar Forecast";

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
		// stack and legend order: loadpoints first, then batteries
		entryOrder(): number[] {
			const batteries: number[] = [];
			const vehicles: number[] = [];
			this.batteryDetails.forEach((d, i) =>
				(d.type === "battery" ? batteries : vehicles).push(i)
			);
			return [...vehicles, ...batteries];
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
			const series: Record<string, unknown>[] = [
				{
					name: GRID_LABEL,
					type: "line",
					z: 4,
					data: this.gridPower,
					showSymbol: false,
					smooth: 0.2,
					lineStyle: { color: colors.grid || "", width: 2 },
					itemStyle: { color: colors.grid || "" },
					emphasis: { disabled: true },
				},
				{
					name: SOLAR_LABEL,
					type: "line",
					z: 4,
					data: this.evopt.req.time_series.ft.map(this.toKW),
					showSymbol: false,
					smooth: 0.2,
					lineStyle: { color: colors.self || "", width: 3 },
					itemStyle: { color: colors.self || "" },
					emphasis: { disabled: true },
				},
				{
					name: this.consumptionLabel,
					type: "bar",
					stack: "charge",
					data: this.evopt.req.time_series.gt.map(this.toKW),
					itemStyle: { color: this.consumptionColor },
					emphasis: { disabled: true },
				},
			];
			this.entryOrder.forEach((index) => {
				const battery = this.evopt.res.batteries[index];
				if (!battery) return;
				// charging positive, discharging negative; one of both is always zero
				const power = battery.charging_power.map((charging, i) => {
					const chargingKW = this.toKW(charging, i);
					const dischargingKW = this.toKW(battery.discharging_power[i] || 0, i);
					return chargingKW > 0 ? chargingKW : -dischargingKW;
				});
				series.push({
					name: this.getBatteryTitle(index),
					type: "bar",
					stack: "charge",
					data: power,
					itemStyle: { color: this.batteryColors[index] },
					emphasis: { disabled: true },
				});
			});
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
					// no chart anchor: category-axis values are scalars, convertToPixel
					// would choke on them; follow the pointer instead
					...tooltipStyle(colors.text || ""),
					formatter: this.tooltipFormatter,
				},
				xAxis: slotXAxis(this.times, this.weekdayShort),
				yAxis: forecastYAxis({
					min: undefined,
					position: "right",
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
				{ label: SOLAR_LABEL, color: colors.self || "", value: "", type: "line" },
				{
					label: this.consumptionLabel,
					color: this.consumptionColor,
					value: "",
					type: "area",
				},
			];
			this.entryOrder.forEach((i) => {
				const detail = this.batteryDetails[i];
				if (!detail) return;
				legends.push({
					label: this.getBatteryTitle(i),
					color: this.batteryColors[i] || "",
					value: "",
					type: "area",
					id: detail.type === "vehicle" ? loadpointTitle(detail) : undefined,
				});
			});
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
		getBatteryTitle(index: number): string {
			const detail = this.batteryDetails[index];
			return detail ? detail.title || detail.name : `Battery ${index + 1}`;
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
