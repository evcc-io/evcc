<template>
	<div>
		<select
			:id="id"
			:value="effectivePredictor"
			class="form-select form-select-sm"
			@change="onChange"
		>
			<option value="daily">{{ $t("main.loadpointSettings.demandPredictor.daily.description") }}</option>
			<option value="weekday">{{ $t("main.loadpointSettings.demandPredictor.weekday.description") }}</option>
			<option value="temperature">{{ $t("main.loadpointSettings.demandPredictor.temperature.description") }}</option>
		</select>
		<div class="mt-1">
			<small v-if="effectivePredictor === 'temperature' && tariffTemperature === undefined" class="text-warning">
				{{ $t("main.loadpointSettings.demandPredictor.noTempTariff") }}
			</small>
			<small v-else class="text-muted">
				{{ $t(`main.loadpointSettings.demandPredictor.${effectivePredictor}.description`) }}
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import api from "@/api";

export default defineComponent({
	name: "DemandPredictorDropdown",
	props: {
		id: { type: String, required: true },
		loadpointId: { type: String, required: true },
		demandPredictor: { type: String, default: "" },
		chargerFeatureDemandTemperature: { type: Boolean, default: false },
		chargerFeatureDemandWeekday: { type: Boolean, default: false },
		tariffTemperature: { type: Number, default: undefined },
	},
	computed: {
		effectivePredictor(): string {
			if (this.demandPredictor) {
				return this.demandPredictor;
			}
			if (this.chargerFeatureDemandTemperature) {
				return "temperature";
			}
			if (this.chargerFeatureDemandWeekday) {
				return "weekday";
			}
			return "daily";
		},
	},
	methods: {
		onChange(event: Event) {
			const value = (event.target as HTMLSelectElement).value;
			api.post(`loadpoints/${this.loadpointId}/demandpredictor/${value}`);
		},
	},
});
</script>
