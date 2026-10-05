<template>
	<LabelAndValue
		class="flex-grow-1"
		:label="$t('main.targetEnergy.label')"
		align="end"
		data-testid="limit-energy"
	>
		<h3 class="value m-0">
			<CustomSelect inline :options="selectOptions" :selected="limitEnergy" @change="change">
				<span
					class="text-decoration-underline"
					:class="{ 'text-gray fw-normal': !limitEnergy }"
					data-testid="limit-energy-value"
				>
					<AnimatedNumber :to="limitEnergy" :format="fmtEnergy" />
				</span>
			</CustomSelect>

			<div v-if="estimated" class="extraValue text-nowrap">
				<AnimatedNumber :to="estimated" :format="fmtSoc" />
			</div>
		</h3>
	</LabelAndValue>
</template>

<script lang="ts">
import LabelAndValue from "../Helper/LabelAndValue.vue";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import formatter from "@/mixins/formatter";
import { estimatedSoc, energyOptions, optionStep, fmtEnergy } from "@/utils/energyOptions.ts";
import { defineComponent } from "vue";

export default defineComponent({
	name: "LimitEnergySelect",
	components: { LabelAndValue, AnimatedNumber, CustomSelect },
	mixins: [formatter],
	props: {
		limitEnergy: { type: Number, default: 0 },
		socPerKwh: Number,
		chargedEnergy: { type: Number, required: true },
		capacity: Number,
	},
	emits: ["limit-energy-updated"],
	computed: {
		options() {
			return energyOptions(
				this.chargedEnergy,
				this.capacity || 100,
				this.fmtWh,
				this.fmtPercentage,
				this.$t("main.targetEnergy.noLimit"),
				this.socPerKwh
			);
		},
		selectOptions() {
			return this.options.map(({ energy, text, disabled }) => ({
				value: energy,
				name: text,
				disabled,
			}));
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
	margin-top: 0.1rem;
	color: var(--evcc-gray);
	font-size: 14px;
	font-weight: normal;
}
</style>
