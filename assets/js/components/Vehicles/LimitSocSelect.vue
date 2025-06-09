<template>
	<LabelAndValue class="flex-grow-1" :label="title" align="end" data-testid="limit-soc">
		<h3 class="value m-0">
			<label class="position-relative" role="button">
				<select :value="limitSoc" class="custom-select" @change="change">
					<option v-for="{ soc, text } in options" :key="soc" :value="soc">
						{{ text }}
					</option>
				</select>
				<span class="text-decoration-underline" data-testid="limit-soc-value">
					<AnimatedNumber :to="limitSoc" :format="formatSoc" />
				</span>
			</label>

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
import { distanceUnit } from "@/units";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "LimitSocSelect",
	components: { LabelAndValue, AnimatedNumber },
	mixins: [formatter],
	props: {
		limitSoc: { type: Number, default: 0 },
		rangePerSoc: Number,
		heating: Boolean,
	},
	emits: ["limit-soc-updated"],
	computed: {
		options() {
			const result = [];
			for (let soc = 20; soc <= 100; soc += this.step) {
				const text = this.fmtSocOption(soc, this.rangePerSoc, distanceUnit(), this.heating);
				result.push({ soc, text });
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
