<template>
	<div class="table-responsive">
		<table class="table table-sm text-nowrap w-auto">
			<thead>
				<tr>
					<th scope="col" class="sticky-col"></th>
					<th
						v-for="hour in hourGroups"
						:key="hour.key"
						scope="col"
						:colspan="hour.span"
						class="text-start hour-start"
					>
						{{ hour.label }}
					</th>
				</tr>
			</thead>
			<tbody>
				<template v-for="(group, groupIndex) in groups" :key="groupIndex">
					<tr v-if="group.title">
						<th scope="row" class="sticky-col group-header">
							<div class="d-flex align-items-center gap-2">
								<span
									v-if="group.color"
									class="battery-indicator"
									:style="{ backgroundColor: group.color }"
								></span>
								<span
									v-else-if="group.dash"
									class="dash"
									:style="{ backgroundColor: group.dash }"
								></span>
								{{ group.title }}
							</div>
						</th>
						<td :colspan="slotCount"></td>
					</tr>
					<tr v-for="row in group.rows" :key="row.label + (row.unit || '')">
						<th scope="row" class="sticky-col fw-normal">
							{{ row.label }}
							<span v-if="row.unit" class="text-muted small">{{ row.unit }}</span>
						</th>
						<td
							v-for="(value, index) in row.display"
							:key="index"
							class="text-end"
							:class="{
								zero: row.nums[index] === 0,
								'hour-start': hourStarts[index],
							}"
							:style="heatStyle(row, index)"
						>
							{{ value }}
						</td>
					</tr>
				</template>
			</tbody>
		</table>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import type { CURRENCY, BatteryDetail } from "@/types/evcc";

export interface EvoptData {
	req: {
		time_series: {
			ft: number[];
			p_E: number[];
			p_N: number[];
			gt: number[];
			dt: number[];
		};
		batteries?: Array<{
			c_min: number;
			c_max: number;
			d_max: number;
		}>;
	};
	res: {
		grid_export: number[];
		grid_import: number[];
		batteries: Array<{
			charging_power: number[];
			discharging_power: number[];
			state_of_charge: number[];
		}>;
	};
}

interface Row {
	label: string;
	unit?: string;
	display: string[];
	nums: number[];
	heatColor?: string;
	aMax?: number;
	ref?: number; // heat reference; defaults to the row's absolute maximum
}

interface RowGroup {
	title?: string;
	color?: string;
	dash?: string;
	rows: Row[];
}

export default defineComponent({
	name: "TimeSeriesDataTable",
	mixins: [formatter],
	props: {
		evopt: {
			type: Object as PropType<EvoptData>,
			required: true,
		},
		mode: {
			type: String as PropType<"request" | "response">,
			required: true,
		},
		batteryDetails: {
			type: Array as PropType<BatteryDetail[]>,
			required: true,
		},
		timestamps: {
			type: Array as PropType<string[]>,
			default: () => [],
		},
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
		batteryColors: {
			type: Array as PropType<string[]>,
			default: () => [],
		},
	},
	computed: {
		slotCount(): number {
			return this.evopt?.req.time_series.dt.length || 0;
		},
		hourStarts(): boolean[] {
			return this.timestamps.map((ts, i) => i > 0 && new Date(ts).getMinutes() === 0);
		},
		// one th per hour, spanning its slots; midnight carries the weekday
		hourGroups(): { key: string; label: string; span: number }[] {
			const groups: { key: string; label: string; span: number }[] = [];
			for (let i = 0; i < this.slotCount; i++) {
				const d = new Date(this.timestamps[i] || 0);
				const key = `${d.getDate()}-${d.getHours()}`;
				const last = groups[groups.length - 1];
				if (last?.key === key) {
					last.span++;
				} else {
					const hour = d.getHours();
					const label = hour === 0 ? `${this.weekdayShort(d)} 0` : String(hour);
					groups.push({ key, label, span: 1 });
				}
			}
			return groups;
		},
		groups(): RowGroup[] {
			return this.mode === "request" ? this.requestGroups() : this.responseGroups();
		},
	},
	methods: {
		requestGroups(): RowGroup[] {
			const ts = this.evopt.req.time_series;
			const priceUnit = this.pricePerKWhUnit(this.currency);
			return [
				{
					rows: [
						this.powerRow("Solar Forecast", ts.ft, colors.self || ""),
						this.powerRow("Household Demand", ts.gt, colors.muted || ""),
						{
							label: "Time Step",
							unit: "h",
							display: ts.dt.map((s) => (s / 3600).toFixed(2)),
							nums: ts.dt,
						},
						this.priceRow("Grid Import", priceUnit, ts.p_N, colors.price || ""),
						this.priceRow("Grid Export", priceUnit, ts.p_E, colors.export || ""),
					],
				},
			];
		},
		responseGroups(): RowGroup[] {
			const res = this.evopt.res;
			const groups: RowGroup[] = [
				{
					title: "Grid",
					dash: colors.grid || "",
					rows: [
						this.powerRow("Grid Export", res.grid_export, colors.export || ""),
						this.powerRow("Grid Import", res.grid_import, colors.muted || ""),
					],
				},
			];
			res.batteries.forEach((battery, i) => {
				const color = this.batteryColors[i] || "";
				const config = this.evopt.req.batteries?.[i];
				const capacity = this.batteryDetails[i]?.capacity || 0;
				groups.push({
					title: this.getBatteryTitle(i),
					color,
					rows: [
						this.powerRow("Charging Power", battery.charging_power, color, {
							ref: (config?.c_max || 0) / 1000,
						}),
						this.powerRow("Discharging Power", battery.discharging_power, color, {
							ref: (config?.d_max || 0) / 1000,
						}),
						{
							label: "State of Charge",
							unit: "kWh",
							display: battery.state_of_charge.map((wh) =>
								this.fmtWh(wh, this.POWER_UNIT.KW, false, 1)
							),
							nums: battery.state_of_charge,
							heatColor: color,
							ref: capacity * 1000,
						},
						{
							label: "State of Charge",
							unit: "%",
							display: battery.state_of_charge.map((wh) =>
								capacity > 0 ? this.fmtNumber((wh / 1000 / capacity) * 100, 0) : "-"
							),
							nums: battery.state_of_charge.map((wh) =>
								capacity > 0 ? (wh / 1000 / capacity) * 100 : 0
							),
							heatColor: color,
							aMax: 0.3,
							ref: 100,
						},
					],
				});
			});
			return groups;
		},
		powerRow(label: string, whValues: number[], color: string, opts: Partial<Row> = {}): Row {
			const dt = this.evopt.req.time_series.dt;
			const nums = whValues.map((wh, i) => wh / ((dt[i] || 1) / 3600) / 1000);
			return {
				label,
				unit: "kW",
				display: nums.map((kw) => this.fmtNumber(kw, 1)),
				nums,
				heatColor: color,
				...opts,
			};
		},
		priceRow(label: string, unit: string, prices: number[], color: string): Row {
			return {
				label,
				unit,
				display: prices.map((p) =>
					this.fmtPricePerKWh(p * 1000, this.currency, false, false)
				),
				nums: prices,
				heatColor: color,
				aMax: 0.4,
			};
		},
		getBatteryTitle(index: number): string {
			const detail = this.batteryDetails[index];
			return detail ? detail.title || detail.name : `Battery ${index + 1}`;
		},
		heatStyle(row: Row, index: number): Record<string, string> | undefined {
			if (!row.heatColor) return undefined;
			const ref = row.ref ?? Math.max(...row.nums.map(Math.abs));
			if (!ref) return undefined;
			const ratio = Math.min(1, Math.abs(row.nums[index] ?? 0) / ref);
			const alpha = 0.06 + (row.aMax ?? 0.5) * ratio;
			const hex = Math.round(alpha * 255)
				.toString(16)
				.padStart(2, "0");
			return { backgroundColor: row.heatColor.trim() + hex };
		},
	},
});
</script>

<style scoped>
.table {
	font-size: 0.8125rem;
}
.table .small {
	font-size: 0.75rem;
}
.table td,
.table th {
	font-variant-numeric: tabular-nums;
	padding: 0.25rem 0.5rem;
	border-color: rgba(147, 148, 158, 0.1);
}
.table thead th {
	color: var(--bs-gray-medium);
	font-weight: normal;
	border-left: 1px solid rgba(147, 148, 158, 0.25);
}
.sticky-col {
	position: sticky;
	left: 0;
	z-index: 1;
	background: var(--evcc-box);
	border-left: none;
	border-right: 1px solid rgba(147, 148, 158, 0.25);
	padding-right: 1rem;
}
.table thead .sticky-col {
	border: none;
}
.table thead th:nth-child(2) {
	border-left: none;
}
.group-header {
	font-weight: bold;
	padding-top: 0.75rem;
}
.hour-start {
	border-left: 1px solid rgba(147, 148, 158, 0.25);
}
.zero {
	color: var(--bs-gray-medium);
}
.battery-indicator {
	width: 1rem;
	height: 1rem;
	border-radius: 50%;
	flex-shrink: 0;
}
.dash {
	width: 1rem;
	height: 2px;
	border-radius: 1px;
	flex-shrink: 0;
}
</style>
