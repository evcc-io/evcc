<template>
	<LabelAndValue
		class="flex-grow-1"
		:label="$t('main.targetEnergy.label')"
		align="end"
		data-testid="limit-energy"
	>
		<h3 class="value m-0">
			<label class="position-relative">
				<select :value="limitEnergy" class="custom-select" @change="change">
					<option
						v-for="{ energy, text, disabled } in options"
						:key="energy"
						:value="energy"
						:disabled="disabled"
					>
						{{ text }}
					</option>
				</select>
				<span
					class="text-decoration-underline"
					:class="{ 'text-gray fw-normal': !limitEnergy }"
					data-testid="limit-energy-value"
				>
					<AnimatedNumber :to="limitEnergy" :format="fmtEnergy" />
				</span>
			</label>

			<div v-if="estimated" class="extraValue text-nowrap">
				<AnimatedNumber :to="estimated" :format="fmtSoc" />
			</div>
		</h3>
	</LabelAndValue>
</template>

<script>
import LabelAndValue from "./LabelAndValue.vue";
import AnimatedNumber from "./AnimatedNumber.vue";
import formatter from "../mixins/formatter";
import { estimatedSoc, energyOptions, optionStep, fmtEnergy } from "../utils/energyOptions";

export default {
	name: "LimitEnergySelect",
	components: { LabelAndValue, AnimatedNumber },
	mixins: [formatter],
	props: {
		limitEnergy: Number,
		socPerKwh: Number,
		chargedEnergy: Number,
		vehicleCapacity: Number,
	},
	emits: ["limit-energy-updated"],
	computed: {
		options: function () {
			return energyOptions(
				this.chargedEnergy,
				this.vehicleCapacity || 100,
				this.socPerKwh,
				this.fmtKWh,
				this.$t("main.targetEnergy.noLimit")
			);
		},
		step() {
			return optionStep(this.vehicleCapacity || 100);
		},
		estimated: function () {
			return estimatedSoc(this.limitEnergy, this.socPerKwh);
		},
	},
	methods: {
		change: function (e) {
			return this.$emit("limit-energy-updated", parseFloat(e.target.value));
		},
		fmtEnergy: function (value) {
			return fmtEnergy(value, this.step, this.fmtKWh, this.$t("main.targetEnergy.noLimit"));
		},
		fmtSoc: function (value) {
			return `+${Math.round(value)}%`;
		},
	},
};
</script>

<style scoped>
.value {
	font-size: 18px;
}
.extraValue {
	color: var(--evcc-gray);
	font-size: 14px;
}
.custom-select {
	left: 0;
	top: 0;
	bottom: 0;
	right: 0;
	position: absolute;
	opacity: 0;
}
</style>
