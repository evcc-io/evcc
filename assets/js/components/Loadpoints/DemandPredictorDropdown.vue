<template>
	<select
		:id="id"
		:value="effectivePredictor"
		class="form-select form-select-sm"
		@change="onChange"
	>
		<option value="daily">
			{{ $t("main.loadpointSettings.demandPredictor.daily.label") }}
		</option>
		<option value="weekday">
			{{ $t("main.loadpointSettings.demandPredictor.weekday.label") }}
		</option>
		<option value="temperature">
			{{ $t("main.loadpointSettings.demandPredictor.temperature.label") }}
		</option>
	</select>
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
