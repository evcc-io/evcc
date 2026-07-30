<template>
	<LabelAndValue class="flex-grow-1" :label="title" align="end" data-testid="limit-soc">
		<h3 class="value m-0">
			<CustomSelect inline :options="options" :selected="limitSoc" @change="change">
				<span class="text-decoration-underline" data-testid="limit-soc-value">
					<AnimatedNumber :to="limitSoc" :format="formatSoc" />
				</span>
			</CustomSelect>

			<div v-if="estimatedTargetRange" class="extraValue text-nowrap">
				<AnimatedNumber :to="estimatedTargetRange" :format="formatKm" />
			</div>
		</h3>
	</LabelAndValue>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import { distanceUnit } from "@/units";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "LimitSocSelect",
	components: { LabelAndValue, AnimatedNumber, CustomSelect },
	mixins: [formatter],
	props: {
		limitSoc: { type: Number, default: 0 },
		rangePerSoc: Number,
		heating: Boolean,
		minTemp: { type: Number, default: 0 },
		maxTemp: { type: Number, default: 0 },
	},
	emits: ["limit-soc-updated"],
	computed: {
		rangeActive() {
			return this.heating && this.maxTemp > this.minTemp;
		},
		options() {
			const result = [];
			const start = this.rangeActive ? this.minTemp : 20;
			const end = this.rangeActive ? this.maxTemp : 100;
			for (let soc = start; soc <= end; soc += this.step) {
				const name = this.fmtSocOption(soc, this.rangePerSoc, distanceUnit(), this.heating);
				result.push({ value: soc, name });
			}
			return result;
		},
		step() {
			return this.heating ? 1 : 5;
		},
		title() {
			return this.heating
				? this.$t("main.vehicle.tempLimit")
				: this.$t("main.vehicle.targetSoc");
		},
		estimatedTargetRange() {
			return this.estimatedRange(this.limitSoc);
		},
	},
	methods: {
		change(e: Event) {
			return this.$emit(
				"limit-soc-updated",
				parseInt((e.target as HTMLSelectElement).value, 10)
			);
		},
		estimatedRange(soc: number) {
			if (this.rangePerSoc) {
				return Math.round(soc * this.rangePerSoc);
			}
			return null;
		},
		formatSoc(value: number) {
			return this.heating ? this.fmtTemperature(value) : this.fmtPercentage(value);
		},
		formatKm(value: number) {
			return `${this.fmtNumber(value, 0)} ${distanceUnit()}`;
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
