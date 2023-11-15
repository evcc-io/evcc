<template>
	<LabelAndValue
		class="flex-grow-1"
		:label="$t('main.targetEnergy.label')"
		align="end"
		data-testid="target-energy"
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
					data-testid="target-energy-value"
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
		maxEnergy: function () {
			return this.vehicleCapacity || 100;
		},
		steps: function () {
			if (this.maxEnergy < 1) return 0.05;
			if (this.maxEnergy < 2) return 0.1;
			if (this.maxEnergy < 5) return 0.25;
			if (this.maxEnergy < 10) return 0.5;
			if (this.maxEnergy < 25) return 1;
			if (this.maxEnergy < 50) return 2;
			return 5;
		},
		options: function () {
			const result = [];
			for (let energy = 0; energy <= this.maxEnergy; energy += this.steps) {
				let text = this.fmtEnergy(energy);
				const disabled = energy < this.chargedEnergy / 1e3 && energy !== 0;
				const soc = this.estimatedSoc(energy);
				if (soc) {
					text += ` (${this.fmtSoc(soc)})`;
				}
				result.push({ energy, text, disabled });
			}
			return result;
		},
		estimated: function () {
			return this.estimatedSoc(this.limitEnergy);
		},
	},
	methods: {
		change: function (e) {
			return this.$emit("limit-energy-updated", parseFloat(e.target.value));
		},
		estimatedSoc: function (kWh) {
			if (this.socPerKwh) {
				return Math.round(kWh * this.socPerKwh);
			}
			return null;
		},
		fmtEnergy: function (value) {
			if (value === 0) {
				return this.$t("main.targetEnergy.noLimit");
			}

			const inKWh = this.steps >= 0.1;
			const digits = inKWh && this.steps < 1 ? 1 : 0;
			return this.fmtKWh(value * 1e3, inKWh, true, digits);
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
