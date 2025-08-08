<template>
	<div class="mb-4">
		<h3 class="fw-normal mb-3">Time Series Data</h3>
		<div class="table-responsive">
			<table class="table table-sm">
				<thead>
					<tr>
						<th scope="col" class="text-nowrap">Data Series</th>
						<th
							v-for="(_, index) in timeSlots"
							:key="index"
							scope="col"
							:class="['text-end']"
						>
							{{ formatHour(index) }}
						</th>
					</tr>
				</thead>
				<tbody>
					<!-- Request Data -->
					<tr class="table-secondary">
						<td colspan="100%" class="fw-bold text-start">Request Data</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Solar Forecast (kW)</td>
						<td
							v-for="(value, index) in evopt.req.time_series.ft"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatPower(value) }}
						</td>
					</tr>

					<tr>
						<td class="fw-medium text-nowrap text-start">
							{{ gridFeedinPriceLabel }}
						</td>
						<td
							v-for="(value, index) in evopt.req.time_series.p_E"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ fmtPricePerKWh(value * 1000, currency, false, false) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">
							{{ gridImportPriceLabel }}
						</td>
						<td
							v-for="(value, index) in evopt.req.time_series.p_N"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ fmtPricePerKWh(value * 1000, currency, false, false) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Household Demand (kW)</td>
						<td
							v-for="(value, index) in evopt.req.time_series.gt"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatPower(value) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Time Step Duration (h)</td>
						<td
							v-for="(value, index) in evopt.req.time_series.dt"
							:key="index"
							:class="['text-end']"
						>
							{{ formatDuration(value) }}
						</td>
					</tr>

					<!-- Response Data -->
					<tr class="table-secondary">
						<td colspan="100%" class="fw-bold text-start">Response Data</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Grid Export (kW)</td>
						<td
							v-for="(value, index) in evopt.res.grid_export"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatPower(value) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Grid Import (kW)</td>
						<td
							v-for="(value, index) in evopt.res.grid_import"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatPower(value) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">⬇ Import / ⬆ Export</td>
						<td
							v-for="(value, index) in evopt.res.flow_direction"
							:key="index"
							:class="['text-end']"
						>
							<span :title="value === 1 ? 'Export to Grid' : 'Import from Grid'">
								{{ value === 1 ? "⬆" : "⬇" }}
							</span>
						</td>
					</tr>

					<!-- Battery Response Data -->
					<template
						v-for="(battery, batteryIndex) in evopt.res.batteries"
						:key="batteryIndex"
					>
						<tr :style="{ backgroundColor: dimmedBatteryColors[batteryIndex] }">
							<td colspan="100%" class="fw-bold text-start">
								Battery {{ batteryIndex + 1 }} Response
							</td>
						</tr>
						<tr>
							<td class="fw-medium text-nowrap text-start">Charging Power (kW)</td>
							<td
								v-for="(value, index) in battery.charging_power"
								:key="index"
								:class="['text-end', { 'text-muted': value === 0 }]"
							>
								{{ formatPower(value) }}
							</td>
						</tr>
						<tr>
							<td class="fw-medium text-nowrap text-start">Discharging Power (kW)</td>
							<td
								v-for="(value, index) in battery.discharging_power"
								:key="index"
								:class="['text-end', { 'text-muted': value === 0 }]"
							>
								{{ formatPower(value) }}
							</td>
						</tr>
						<tr>
							<td class="fw-medium text-nowrap text-start">State of Charge (kWh)</td>
							<td
								v-for="(value, index) in battery.state_of_charge"
								:key="index"
								:class="['text-end', { 'text-muted': value === 0 }]"
							>
								{{ formatEnergy(value) }}
							</td>
						</tr>
					</template>
				</tbody>
			</table>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { CURRENCY } from "@/types/evcc";

export interface EvoptData {
	req: {
		time_series: {
			ft: number[];
			p_E: number[];
			p_N: number[];
			gt: number[];
			dt: number[];
		};
	};
	res: {
		grid_export: number[];
		grid_import: number[];
		flow_direction: number[];
		batteries: Array<{
			charging_power: number[];
			discharging_power: number[];
			state_of_charge: number[];
		}>;
	};
}

export default defineComponent({
	name: "TimeSeriesDataTable",
	mixins: [formatter],
	props: {
		evopt: {
			type: Object as PropType<EvoptData>,
			required: true,
		},
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
		batteryColors: {
			type: Array as PropType<string[]>,
			required: true,
		},
		dimmedBatteryColors: {
			type: Array as PropType<string[]>,
			required: true,
		},
	},
	computed: {
		timeSlots() {
			// Use the dt array length to determine number of time slots
			return this.evopt?.req.time_series.dt || [];
		},
		gridFeedinPriceLabel() {
			return `Grid Feedin (${this.pricePerKWhUnit(this.currency)})`;
		},
		gridImportPriceLabel() {
			return `Grid Import (${this.pricePerKWhUnit(this.currency)})`;
		},
	},
	methods: {
		formatPower(watts: number): string {
			return this.fmtW(watts, this.POWER_UNIT.KW, false, 1);
		},
		formatEnergy(wh: number): string {
			return this.fmtWh(wh, this.POWER_UNIT.KW, false, 1);
		},
		formatDuration: (seconds: number): string => {
			return (seconds / 3600).toFixed(1);
		},
		formatHour: (index: number): string => {
			const hour = index % 24;
			return hour.toString();
		},
	},
});
</script>

<style scoped>
.table td,
.table th {
	font-variant-numeric: tabular-nums;
}

.table td:not(:first-child) {
	padding-left: 1rem;
}
</style>
