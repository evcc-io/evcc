<template>
	<section class="solar-min-soc border-top px-3 px-sm-4 py-4">
		<h3 class="h5 mb-2">{{ $t("config.tariff.solarMinSoc.title") }}</h3>
		<p class="text-gray mb-4">{{ $t("config.tariff.solarMinSoc.description") }}</p>

		<p v-if="loading" class="text-gray mb-0">{{ $t("config.general.templateLoading") }}</p>
		<template v-else>
			<div class="form-check form-switch mb-4">
				<input
					id="solarMinSocEnabled"
					v-model="config.enabled"
					class="form-check-input"
					type="checkbox"
				/>
				<label class="form-check-label" for="solarMinSocEnabled">
					{{ $t("config.tariff.solarMinSoc.enabled") }}
				</label>
			</div>

			<div class="row g-3 mb-4">
				<div class="col-sm-6">
					<label class="form-label" for="solarMinSocLowThreshold">
						{{ $t("config.tariff.solarMinSoc.lowThreshold") }}
					</label>
					<div class="input-group">
						<input
							id="solarMinSocLowThreshold"
							v-model.number="config.lowThreshold"
							class="form-control"
							type="number"
							min="0"
							step="0.1"
							required
						/>
						<span class="input-group-text">kWh</span>
					</div>
				</div>
				<div class="col-sm-6">
					<label class="form-label" for="solarMinSocMediumThreshold">
						{{ $t("config.tariff.solarMinSoc.mediumThreshold") }}
					</label>
					<div class="input-group">
						<input
							id="solarMinSocMediumThreshold"
							v-model.number="config.mediumThreshold"
							class="form-control"
							type="number"
							:min="config.lowThreshold"
							step="0.1"
							required
						/>
						<span class="input-group-text">kWh</span>
					</div>
				</div>
			</div>

			<div v-if="config.enabled" class="alert alert-light py-2 mb-4" role="status">
				<span v-if="status.available">
					{{
						$t("config.tariff.solarMinSoc.current", {
							energy: status.forecastEnergy.toFixed(1),
							state: $t(`config.tariff.solarMinSoc.state.${status.state}`),
						})
					}}
				</span>
				<span v-else>{{ $t("config.tariff.solarMinSoc.unavailable") }}</span>
			</div>

			<div v-if="status.availableVehicles.length" class="table-responsive mb-4">
				<table class="table align-middle mb-0">
					<thead>
						<tr>
							<th scope="col">{{ $t("config.tariff.solarMinSoc.vehicle") }}</th>
							<th v-for="state in states" :key="state" scope="col">
								{{ $t(`config.tariff.solarMinSoc.state.${state}`) }}
							</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="vehicle in status.availableVehicles" :key="vehicle.name">
							<th scope="row">{{ vehicle.title }}</th>
							<td v-for="state in states" :key="state">
								<div class="input-group input-group-sm soc-input">
									<input
										:id="`solarMinSoc-${vehicle.name}-${state}`"
										v-model.number="config.vehicles[vehicle.name][state]"
										:aria-label="`${vehicle.title}: ${$t(`config.tariff.solarMinSoc.state.${state}`)}`"
										class="form-control"
										type="number"
										min="0"
										max="100"
										step="5"
										required
									/>
									<span class="input-group-text">%</span>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>

			<div v-if="validationError" class="invalid-feedback d-block mb-3">
				{{ validationError }}
			</div>
			<button type="button" class="btn btn-primary" :disabled="saving" @click="save">
				{{ $t("config.general.save") }}
			</button>
		</template>
	</section>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import api from "@/api";
import type { SolarMinSocStatus } from "@/types/evcc";

type ForecastState = NonNullable<SolarMinSocStatus["state"]>;

const emptyStatus = (): SolarMinSocStatus => ({
	enabled: false,
	lowThreshold: 5,
	mediumThreshold: 15,
	vehicles: {},
	availableVehicles: [],
	available: false,
	forecastEnergy: 0,
});

export default defineComponent({
	name: "SolarMinSocConfig",
	data() {
		return {
			loading: true,
			saving: false,
			status: emptyStatus(),
			config: emptyStatus(),
			states: ["low", "medium", "high"] as ForecastState[],
		};
	},
	computed: {
		validationError(): string {
			if (
				this.config.lowThreshold < 0 ||
				this.config.lowThreshold >= this.config.mediumThreshold
			) {
				return this.$t("config.tariff.solarMinSoc.thresholdError");
			}
			for (const values of Object.values(this.config.vehicles)) {
				if (this.states.some((state) => values[state] < 0 || values[state] > 100)) {
					return this.$t("config.tariff.solarMinSoc.socError");
				}
			}
			return "";
		},
	},
	mounted() {
		this.load();
	},
	methods: {
		async load() {
			this.loading = true;
			try {
				const { data } = await api.get<SolarMinSocStatus>("config/solar-min-soc");
				for (const vehicle of data.availableVehicles) {
					data.vehicles[vehicle.name] ||= { low: 0, medium: 0, high: 0 };
				}
				this.status = data;
				this.config = structuredClone(data);
			} finally {
				this.loading = false;
			}
		},
		async save() {
			if (this.validationError) return;
			this.saving = true;
			try {
				const { data } = await api.put<SolarMinSocStatus>("config/solar-min-soc", {
					enabled: this.config.enabled,
					lowThreshold: this.config.lowThreshold,
					mediumThreshold: this.config.mediumThreshold,
					vehicles: this.config.vehicles,
				});
				this.status = data;
				this.config = structuredClone(data);
			} finally {
				this.saving = false;
			}
		},
	},
});
</script>

<style scoped>
.solar-min-soc {
	background: var(--evcc-background);
}

.soc-input {
	min-width: 7rem;
}
</style>
