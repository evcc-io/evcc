<template>
	<div class="form-check form-switch mb-4">
		<input
			id="demandPredictorEnabled"
			:checked="enabled"
			class="form-check-input"
			type="checkbox"
			role="switch"
			@change="toggle"
		/>
		<div class="form-check-label">
			<label for="demandPredictorEnabled">
				{{ $t("config.meter.demandPredictor.label") }}
			</label>
			<div v-if="enabled" class="mt-3">
				<select
					id="demandPredictor"
					class="form-select"
					:value="predictor"
					:aria-label="$t('config.loadpoint.demandPredictor.label')"
					@change="change"
				>
					<option v-for="option in options" :key="option.value" :value="option.value">
						{{ option.name }}
					</option>
				</select>
				<div class="form-text evcc-gray">{{ help }}</div>
				<div v-if="temperatureTariffMissing" class="form-text text-warning">
					{{ $t("config.loadpoint.demandPredictor.noTempTariff") }}
					<a href="#" @click.prevent="$emit('add-temperature-tariff')">
						{{ $t("config.loadpoint.demandPredictor.noTempTariffLink") }}
					</a>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { DemandPredictor, DevicePreferences, SelectOption } from "@/types/evcc";

const PREDICTORS: DemandPredictor[] = ["weekday", "temperature"];

export default defineComponent({
	name: "DemandPredictorSwitch",
	props: {
		modelValue: Object as PropType<DevicePreferences>,
		temperatureTariffConfigured: Boolean,
	},
	emits: ["update:modelValue", "add-temperature-tariff"],
	computed: {
		predictor(): DemandPredictor | "" {
			return this.modelValue?.demandPredictor ?? "";
		},
		enabled(): boolean {
			return !!this.predictor;
		},
		options(): SelectOption<DemandPredictor>[] {
			return PREDICTORS.map((value) => ({
				value,
				name: this.$t(`config.loadpoint.demandPredictor.${value}.label`),
			}));
		},
		help(): string {
			return this.$t(`config.loadpoint.demandPredictor.${this.predictor}.description`);
		},
		temperatureTariffMissing(): boolean {
			return this.predictor === "temperature" && !this.temperatureTariffConfigured;
		},
	},
	methods: {
		update(demandPredictor: string) {
			this.$emit("update:modelValue", { ...this.modelValue, demandPredictor });
		},
		toggle() {
			this.update(this.enabled ? "" : PREDICTORS[0]!);
		},
		change(event: Event) {
			this.update((event.target as HTMLSelectElement).value);
		},
	},
});
</script>
