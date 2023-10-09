<template>
	<LabelAndValue
		class="flex-grow-1"
		:label="$t('main.targetEnergy.label')"
		align="end"
		data-testid="target-energy"
	>
		<h3 class="value m-0 d-block d-sm-flex align-items-baseline justify-content-end">
			<label class="position-relative">
				<select :value="targetEnergy" class="custom-select" @change="change">
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
					:class="{ 'text-gray fw-normal': !targetEnergy }"
					data-testid="target-energy-value"
				>
					<AnimatedNumber :to="targetEnergy" :format="fmtEnergy" />
				</span>
			</label>

			<div v-if="estimatedTargetSoc" class="extraValue ms-0 ms-sm-1 text-nowrap">
				<AnimatedNumber :to="estimatedTargetSoc" :format="fmtSoc" />
			</div>
		</h3>
	</LabelAndValue>
</template>

<script>
import LabelAndValue from "./LabelAndValue.vue";
import AnimatedNumber from "./AnimatedNumber.vue";
import formatter from "../mixins/formatter";

export default {
	name: "TargetEnergySelect",
	components: { LabelAndValue, AnimatedNumber },
	mixins: [formatter],
	props: {
		targetEnergy: Number,
		socPerKwh: Number,
		chargedEnergy: Number,
		vehicleCapacity: Number,
	},
	emits: ["target-energy-updated"],
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
		estimatedTargetSoc: function () {
			return this.estimatedSoc(this.targetEnergy);
		},
	},
	methods: {
		change: function (e) {
			return this.$emit("target-energy-updated", parseFloat(e.target.value));
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
