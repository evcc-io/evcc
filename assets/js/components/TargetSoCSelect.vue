<template>
	<LabelAndValue class="flex-grow-1" :label="$t('main.vehicle.targetSoC')" align="end">
		<h3 class="value m-0 d-block d-sm-flex align-items-baseline justify-content-end">
			<label class="position-relative">
				<select :value="targetSoc" class="custom-select" @change="change">
					<option v-for="{ soc, text } in options" :key="soc" :value="soc">
						{{ text }}
					</option>
				</select>
				<span class="text-decoration-underline">
					<AnimatedNumber :to="targetSoc" :format="formatSoC" />
				</span>
			</label>

			<div v-if="estimatedTargetRange" class="extraValue ms-0 ms-sm-1 text-nowrap">
				<AnimatedNumber :to="estimatedTargetRange" :format="formatKm" />
			</div>
		</h3>
	</LabelAndValue>
</template>

<script>
import LabelAndValue from "./LabelAndValue.vue";
import AnimatedNumber from "./AnimatedNumber.vue";
import { distanceUnit } from "../units";

export default {
	name: "TargetSoCSelect",
	components: { LabelAndValue, AnimatedNumber },
	props: {
		targetSoc: Number,
		rangePerSoc: Number,
	},
	emits: ["target-soc-updated"],

	computed: {
		options: function () {
			const result = [];
			for (let soc = 20; soc <= 100; soc += 5) {
				let text = this.formatSoC(soc);
				const range = this.estimatedRange(soc);
				if (range) {
					text += ` (${this.formatKm(range)})`;
				}
				result.push({ soc, text });
			}
			return result;
		},
		estimatedTargetRange: function () {
			return this.estimatedRange(this.targetSoc);
		},
	},
	methods: {
		change: function (e) {
			return this.$emit("target-soc-updated", parseInt(e.target.value, 10));
		},
		estimatedRange: function (soc) {
			if (this.rangePerSoc) {
				return Math.round(soc * this.rangePerSoc);
			}
			return null;
		},
		formatSoC: function (value) {
			return `${Math.round(value)}%`;
		},
		formatKm: function (value) {
			return `${Math.round(value)} ${distanceUnit()}`;
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
