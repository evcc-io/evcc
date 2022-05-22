<template>
	<LabelAndValue class="flex-grow-1" :label="$t('main.vehicle.targetSoC')" :on-dark="true">
		<h3 class="value m-0">
			<label class="d-inline-block position-relative">
				<select :value="targetSoc" class="custom-select" @change="change">
					<option v-for="{ soc, text } in options" :key="soc" :value="soc">
						{{ text }}
					</option>
				</select>
				<span class="text-decoration-underline">{{ targetSoc }}%</span>
			</label>

			<span v-if="estimatedTargetRange" class="extraValue d-block d-sm-inline text-nowrap">
				&nbsp;{{ estimatedTargetRange }}km
			</span>
		</h3>
	</LabelAndValue>
</template>

<script>
import LabelAndValue from "./LabelAndValue.vue";

export default {
	name: "TargetSoCSelect",
	components: { LabelAndValue },
	props: {
		targetSoc: Number,
		rangePerSoc: Number,
	},
	emits: ["target-soc-updated"],

	computed: {
		options: function () {
			const result = [];
			for (let soc = 20; soc <= 100; soc += 5) {
				let text = `${soc}%`;
				const range = this.estimatedRange(soc);
				if (range) {
					text += ` (${range}km)`;
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
			return this.$emit("target-soc-updated", e.target.value);
		},
		estimatedRange: function (soc) {
			if (this.rangePerSoc) {
				return Math.round(soc * this.rangePerSoc);
			}
			return null;
		},
	},
};
</script>

<style scoped>
.value {
	font-size: 18px;
}
.extraValue {
	color: var(--bs-gray-light);
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
