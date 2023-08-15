<template>
	<LabelAndValue class="flex-grow-1" :label="title" align="end" data-testid="target-soc">
		<h3 class="value m-0 d-block d-sm-flex align-items-baseline justify-content-end">
			<label class="position-relative">
				<select :value="targetSoc" class="custom-select" @change="change">
					<option v-for="{ soc, text } in options" :key="soc" :value="soc">
						{{ text }}
					</option>
				</select>
				<span class="text-decoration-underline" data-testid="target-soc-value">
					<AnimatedNumber :to="targetSoc" :format="formatSoc" />
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
import formatter from "../mixins/formatter";

export default {
	name: "TargetSocSelect",
	components: { LabelAndValue, AnimatedNumber },
	mixins: [formatter],
	props: {
		targetSoc: Number,
		rangePerSoc: Number,
		heating: Boolean,
	},
	emits: ["target-soc-updated"],

	computed: {
		options: function () {
			const result = [];
			for (let soc = 20; soc <= 100; soc += 5) {
				let text = this.formatSoc(soc);
				const range = this.estimatedRange(soc);
				if (range) {
					text += ` (${this.formatKm(range)})`;
				}
				result.push({ soc, text });
			}
			return result;
		},
		title: function () {
			return this.heating
				? this.$t("main.vehicle.tempLimit")
				: this.$t("main.vehicle.targetSoc");
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
		formatSoc: function (value) {
			// todo: add fahrenheit support
			return this.heating ? this.fmtNumber(value, 1, "celsius") : `${Math.round(value)}%`;
		},
		formatKm: function (value) {
			return `${this.fmtNumber(value, 0)} ${distanceUnit()}`;
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
