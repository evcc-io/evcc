<template>
	<div class="mb-4">
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
						<td :colspan="timeSlots.length + 1" class="fw-bold text-start">
							Request Data
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Solar Forecast (kW)</td>
						<td
							v-for="(value, index) in evopt.req.time_series.ft"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatEnergyToPower(value, index) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Household Demand (kW)</td>
						<td
							v-for="(value, index) in evopt.req.time_series.gt"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatEnergyToPower(value, index) }}
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

					<!-- Response Data -->
					<tr class="table-secondary">
						<td :colspan="timeSlots.length + 1" class="fw-bold text-start">
							Response Data
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Grid Export (kW)</td>
						<td
							v-for="(value, index) in evopt.res.grid_export"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatEnergyToPower(value, index) }}
						</td>
					</tr>
					<tr>
						<td class="fw-medium text-nowrap text-start">Grid Import (kW)</td>
						<td
							v-for="(value, index) in evopt.res.grid_import"
							:key="index"
							:class="['text-end', { 'text-muted': value === 0 }]"
						>
							{{ formatEnergyToPower(value, index) }}
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
						<tr>
							<td :colspan="timeSlots.length + 1" class="fw-bold text-start">
								<div class="d-flex align-items-center">
									<span
										class="battery-indicator me-2"
										:style="{ backgroundColor: batteryColors[batteryIndex] }"
									></span>
									{{ getBatteryTitle(batteryIndex) }}
								</div>
							</td>
						</tr>
						<tr>
							<td class="fw-medium text-nowrap text-start">Charging Power (kW)</td>
							<td
								v-for="(value, index) in battery.charging_power"
								:key="index"
								:class="['text-end', { 'text-muted': value === 0 }]"
							>
								{{ formatEnergyToPower(value, index) }}
							</td>
						</tr>
						<tr>
							<td class="fw-medium text-nowrap text-start">Discharging Power (kW)</td>
							<td
								v-for="(value, index) in battery.discharging_power"
								:key="index"
								:class="['text-end', { 'text-muted': value === 0 }]"
							>
								{{ formatEnergyToPower(value, index) }}
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
						<tr>
							<td class="fw-medium text-nowrap text-start">State of Charge (%)</td>
							<td
								v-for="(value, index) in battery.state_of_charge"
								:key="index"
								:class="['text-end', { 'text-muted': value === 0 }]"
							>
								{{ formatSocPercentage(value, batteryIndex) }}
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
		batteryDetails: {
			type: Array as PropType<BatteryDetail[]>,
			required: true,
		},
		timestamp: {
			type: String,
			default: "",
		},
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
		batteryColors: {
			type: Array as PropType<string[]>,
			default: () => [],
		},
		dimmedBatteryColors: {
			type: Array as PropType<string[]>,
			default: () => [],
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
		formatEnergyToPower(wh: number, index: number): string {
			// Convert Wh to kW by normalizing against time duration
			const dtSeconds = this.evopt.req.time_series.dt[index] || 0;
			const hours = dtSeconds / 3600; // Convert seconds to hours
			const watts = wh / hours; // Convert to W (power)
			return this.fmtW(watts, this.POWER_UNIT.KW, false, 1);
		},
		formatEnergy(wh: number): string {
			return this.fmtWh(wh, this.POWER_UNIT.KW, false, 1);
		},
		formatDuration: (seconds: number): string => {
			return (seconds / 3600).toFixed(2);
		},
		formatHour(index: number): string {
			// Show label only every 4th slot (every hour for 15-minute slots)
			if (index % 4 !== 0) {
				return "";
			}
			const startTime = new Date(this.timestamp);
			const currentTime = new Date(startTime.getTime() + index * 15 * 60 * 1000); // Add 15-minute intervals
			return currentTime.getHours().toString();
		},
		getBatteryTitle(index: number): string {
			const detail = this.batteryDetails[index];
			return detail ? detail.title || detail.name : `Battery ${index + 1}`;
		},
		formatSocPercentage(socWh: number, batteryIndex: number): string {
			const detail = this.batteryDetails[batteryIndex];
			if (detail?.capacity && detail.capacity > 0) {
				const percentage = (socWh / 1000 / detail.capacity) * 100;
				return this.fmtNumber(percentage, 0);
			}
			return "-";
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

.battery-indicator {
	width: 1rem;
	height: 1rem;
	border-radius: 50%;
	flex-shrink: 0;
}
</style>
