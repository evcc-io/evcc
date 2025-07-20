<template>
	<div class="root">
		<div class="mb-2 label text-truncate-xs-only" :class="labelClass">
			<slot name="label">{{ label }}</slot>
		</div>
		<slot>
			<h3 class="value m-0" :class="valueClass">
				<slot name="value">
					<AnimatedNumber
						v-if="valueFmt && typeof value === 'number'"
						:to="value"
						:format="valueFmt"
					/>
					<span v-else>{{ value }}</span>
					<div v-if="extraValue != null" class="extraValue text-nowrap">
						{{ extraValue || "&nbsp;" }}
					</div>
				</slot>
			</h3>
		</slot>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import AnimatedNumber from "./AnimatedNumber.vue";

export default defineComponent({
	name: "LabelAndValue",
	components: { AnimatedNumber },
	props: {
		label: String,
		value: [Number, String],
		valueFmt: Function as PropType<(n: number) => string>,
		extraValue: String,
		align: { type: String, default: "center" },
	},
	computed: {
		labelClass() {
			return `text-${this.align}`;
		},
		valueClass() {
			return `justify-content-${this.align}`;
		},
	},
});
</script>
<style scoped>
.root {
	margin-bottom: 1rem;
}
.label {
	text-transform: uppercase;
	color: var(--evcc-gray);
	font-size: 14px;
}
.value {
	font-size: 18px;
}
.extraValue {
	color: var(--evcc-gray);
	font-size: 14px;
}
</style>
