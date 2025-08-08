<template>
	<div class="mb-4">
		<h3 class="fw-normal mb-3">Battery Configuration</h3>
		<div class="table-responsive">
			<table class="table">
				<thead>
					<tr>
						<th scope="col">#</th>
						<th scope="col">Power Range (kW)</th>
						<th scope="col">Max Discharge (kW)</th>
						<th scope="col">SoC Range (kWh)</th>
						<th scope="col">Initial SoC (kWh)</th>
						<th scope="col">Energy Value (€/kWh)</th>
						<th scope="col">Grid Interaction</th>
						<th scope="col">Demand Profile</th>
						<th scope="col">SoC Goals</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="(battery, index) in batteries" :key="index">
						<th scope="row">{{ index + 1 }}</th>
						<td>
							{{ formatPowerRange(battery.c_min, battery.c_max) }}
						</td>
						<td>{{ formatPower(battery.d_max) }}</td>
						<td>
							{{ formatEnergyRange(battery.s_min, battery.s_max) }}
						</td>
						<td>{{ formatEnergy(battery.s_initial) }}</td>
						<td>
							{{ fmtMoney(battery.p_a * 1000, currency, true, true) }}
						</td>
						<td>
							<div class="d-flex flex-column gap-1">
								<span v-if="battery.charge_from_grid" class="badge bg-primary">
									Grid Charge
								</span>
								<span v-if="battery.discharge_to_grid" class="badge bg-success">
									Grid Discharge
								</span>
								<span
									v-if="!battery.charge_from_grid && !battery.discharge_to_grid"
									class="text-muted"
								>
									No Grid Interaction
								</span>
							</div>
						</td>
						<td>
							<span v-if="battery.p_demand?.length" class="badge bg-info">
								{{ battery.p_demand.length }} steps
							</span>
							<span v-else class="text-muted">None</span>
						</td>
						<td>
							<span v-if="battery.s_goal?.length" class="badge bg-warning">
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
import type { CURRENCY } from "@/types/evcc";

export interface BatteryConfig {
	c_min: number;
	c_max: number;
	d_max: number;
	s_min: number;
	s_max: number;
	s_initial: number;
	p_a: number;
	charge_from_grid: boolean;
	discharge_to_grid: boolean;
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
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
	},
	methods: {
		formatPower(watts: number): string {
			return this.fmtW(watts, this.POWER_UNIT.KW, false, 1);
		},
		formatEnergy(wh: number): string {
			return this.fmtWh(wh, this.POWER_UNIT.KW, false, 1);
		},
		formatPowerRange(min: number, max: number): string {
			return `${this.formatPower(min)} - ${this.formatPower(max)}`;
		},
		formatEnergyRange(min: number, max: number): string {
			return `${this.formatEnergy(min)} - ${this.formatEnergy(max)}`;
		},
	},
});
</script>

<style scoped>
.table td,
.table th {
	font-variant-numeric: tabular-nums;
}
</style>
