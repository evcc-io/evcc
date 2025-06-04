<template>
	<LabelAndValue
		class="flex-grow-1"
		:label="$t('main.targetEnergy.label')"
		align="end"
		data-testid="limit-energy"
	>
		<h3 class="value m-0">
			<label class="position-relative" role="button">
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

<script lang="ts">
import LabelAndValue from "../Helper/LabelAndValue.vue";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import formatter from "@/mixins/formatter";
import { estimatedSoc, energyOptions, optionStep, fmtEnergy } from "@/utils/energyOptions";
import { defineComponent } from "vue";

export default defineComponent({
	name: "LimitEnergySelect",
	components: { LabelAndValue, AnimatedNumber },
	mixins: [formatter],
	props: {
		limitEnergy: Number,
		socPerKwh: Number,
		chargedEnergy: Number,
		capacity: Number,
	},
	emits: ["limit-energy-updated"],
	computed: {
		options() {
			return energyOptions(
				this.chargedEnergy,
				this.capacity || 100,
				this.socPerKwh,
				this.fmtWh,
				this.fmtPercentage,
				this.$t("main.targetEnergy.noLimit")
			);
		},
		step() {
			return optionStep(this.capacity || 100);
		},
		estimated() {
			return estimatedSoc(this.limitEnergy, this.socPerKwh);
		},
	},
	methods: {
		change(e: Event) {
			return this.$emit(
				"limit-energy-updated",
				parseFloat((e.target as HTMLSelectElement).value)
			);
		},
		fmtEnergy(value: number) {
			return fmtEnergy(value, this.step, this.fmtWh, this.$t("main.targetEnergy.noLimit"));
		},
		fmtSoc(value: number) {
			return `+${this.fmtPercentage(value)}`;
		},
	},
});
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
	cursor: pointer;
	position: absolute;
	opacity: 0;
}
</style>
