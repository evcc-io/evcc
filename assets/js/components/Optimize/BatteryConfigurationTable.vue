<template>
	<div>
		<div class="table-responsive">
			<table class="table table-sm align-top w-auto text-nowrap">
				<thead>
					<tr>
						<th scope="col">Battery</th>
						<th scope="col">State of Charge</th>
						<th scope="col">SoC Range</th>
						<th scope="col">Energy Value</th>
						<th scope="col">Power Range</th>
						<th scope="col">Max Discharge</th>
						<th scope="col">Grid Interaction</th>
						<th scope="col">Demand</th>
						<th scope="col">SoC Goals</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="(battery, index) in batteries" :key="index">
						<th scope="row">
							<div class="d-flex align-items-center gap-2">
								<span
									class="battery-indicator"
									:style="{ backgroundColor: batteryColors[index] }"
								></span>
								{{ getBatteryTitle(index) }}
							</div>
							<div class="progress soc-progress mt-2">
								<div
									class="progress-bar"
									:style="{
										width: socPercentage(battery.s_initial, index) + '%',
										backgroundColor: batteryColors[index],
									}"
								></div>
							</div>
						</th>
						<td>
							<div>{{ formatStateOfCharge(battery.s_initial, index) }}</div>
							<div class="text-muted small">
								{{ formatInitialSocPercentage(battery.s_initial, index) }}
							</div>
						</td>
						<td>
							<div>{{ formatEnergyRange(battery.s_min, battery.s_max) }}</div>
							<div class="text-muted small">
								{{ formatSocRangePercentage(battery.s_min, battery.s_max, index) }}
							</div>
						</td>
						<td>
							<div>{{ formatEnergyValue(battery.p_a) }}</div>
							<div class="text-muted small">
								{{ formatTotalEnergyValue(battery.p_a, index) }} total
							</div>
						</td>
						<td>
							{{ formatPowerRange(battery.c_min, battery.c_max) }}
						</td>
						<td>{{ formatPower(battery.d_max) }}</td>
						<td>
							<span :class="{ 'text-muted': !hasGridInteraction(battery) }">
								{{ formatGridInteraction(battery) }}
							</span>
						</td>
						<td>
							<span v-if="battery.p_demand?.length">
								{{ battery.p_demand.length }} steps
							</span>
							<span v-else class="text-muted">None</span>
						</td>
						<td>
							<span
								v-if="battery.s_goal?.length"
								class="badge rounded-pill text-bg-light"
							>
								{{ battery.s_goal.length }} goals
							</span>
							<span v-else class="text-muted">None</span>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { CURRENCY, BatteryDetail } from "@/types/evcc";

export interface BatteryConfig {
	c_min: number;
	c_max: number;
	d_max: number;
	s_min: number;
	s_max: number;
	s_initial: number;
	p_a: number;
	charge_from_grid?: boolean;
	discharge_to_grid?: boolean;
	p_demand?: number[];
	s_goal?: number[];
}

export default defineComponent({
	name: "BatteryConfigurationTable",
	mixins: [formatter],
	props: {
		batteries: {
			type: Array as PropType<BatteryConfig[]>,
			required: true,
		},
		batteryDetails: {
			type: Array as PropType<BatteryDetail[]>,
			required: true,
		},
		batteryColors: {
			type: Array as PropType<string[]>,
			default: () => [],
		},
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
	},
	methods: {
		formatPower(watts: number): string {
			return this.fmtW(watts, this.POWER_UNIT.KW, true, 1);
		},
		formatPowerRange(min: number, max: number): string {
			const minValue = this.fmtW(min, this.POWER_UNIT.KW, false, 1);
			const maxValue = this.fmtW(max, this.POWER_UNIT.KW, true, 1);
			return `${minValue} – ${maxValue}`;
		},
		formatEnergyRange(min: number, max: number): string {
			const minValue = this.fmtWh(min, this.POWER_UNIT.KW, false, 1);
			const maxValue = this.fmtWh(max, this.POWER_UNIT.KW, true, 1);
			return `${minValue} – ${maxValue}`;
		},
		formatEnergyValue(valuePerWh: number): string {
			return this.fmtPricePerKWh(valuePerWh * 1000, this.currency, false, true);
		},
		getBatteryTitle(index: number): string {
			const detail = this.batteryDetails[index];
			return detail ? detail.title || detail.name : `Battery ${index + 1}`;
		},
		socPercentage(initialSocWh: number, index: number): number {
			const detail = this.batteryDetails[index];
			if (detail?.capacity && detail.capacity > 0) {
				return (initialSocWh / 1000 / detail.capacity) * 100;
			}
			return 0;
		},
		formatStateOfCharge(initialSocWh: number, index: number): string {
			const detail = this.batteryDetails[index];
			if (detail?.capacity) {
				const initialSocKWh = this.fmtWh(initialSocWh, this.POWER_UNIT.KW, false, 1);
				const capacityKWh = this.fmtWh(detail.capacity * 1000, this.POWER_UNIT.KW, true, 1);
				return `${initialSocKWh} of ${capacityKWh}`;
			}
			return "-";
		},
		formatInitialSocPercentage(initialSocWh: number, index: number): string {
			return this.fmtPercentage(this.socPercentage(initialSocWh, index), 0);
		},
		formatSocRangePercentage(minSocWh: number, maxSocWh: number, index: number): string {
			const detail = this.batteryDetails[index];
			if (detail?.capacity && detail.capacity > 0) {
				const minPercentage = (minSocWh / 1000 / detail.capacity) * 100;
				const maxPercentage = (maxSocWh / 1000 / detail.capacity) * 100;
				return `${this.fmtPercentage(minPercentage, 0)} – ${this.fmtPercentage(maxPercentage, 0)}`;
			}
			return "";
		},
		formatTotalEnergyValue(valuePerWh: number, index: number): string {
			const detail = this.batteryDetails[index];
			if (detail?.capacity && detail.capacity > 0) {
				const totalValue = valuePerWh * detail.capacity * 1000; // Convert kWh to Wh for calculation
				return this.fmtMoney(totalValue, this.currency, true, true);
			}
			return "";
		},
		hasGridInteraction(battery: BatteryConfig): boolean {
			return !!(battery.charge_from_grid || battery.discharge_to_grid);
		},
		formatGridInteraction(battery: BatteryConfig): string {
			const canCharge = battery.charge_from_grid;
			const canDischarge = battery.discharge_to_grid;

			if (canCharge && canDischarge) {
				return "Charge / Discharge";
			} else if (canCharge) {
				return "Charge";
			} else if (canDischarge) {
				return "Discharge";
			} else {
				return "None";
			}
		},
	},
});
</script>

<style scoped>
.table {
	font-size: 0.8125rem;
}
.table td,
.table th {
	font-variant-numeric: tabular-nums;
	padding-right: 1.75rem;
	padding-top: 0.625rem;
	padding-bottom: 0.625rem;
	border-color: rgba(147, 148, 158, 0.1);
}
.table .small {
	font-size: 0.75rem;
}
.table thead th {
	color: var(--bs-gray-medium);
	font-size: 0.6875rem;
	font-weight: normal;
	text-transform: uppercase;
	letter-spacing: 0.03em;
	padding-top: 0;
	padding-bottom: 0.25rem;
	border-color: rgba(147, 148, 158, 0.25);
}
.battery-indicator {
	width: 0.625rem;
	height: 0.625rem;
	border-radius: 50%;
	flex-shrink: 0;
}
.badge {
	font-size: 0.75rem;
}
.soc-progress {
	height: 0.25rem;
}
.table .text-muted {
	color: var(--bs-gray-medium) !important;
}
</style>
