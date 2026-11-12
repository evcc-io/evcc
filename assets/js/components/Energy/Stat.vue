<template>
	<!-- the wrapper is the query container, rules cannot address the container itself -->
	<div :class="{ 'container-wrapper': !inline }">
		<div
			class="stat"
			:class="{
				'stat--compact': inline,
				'stat--inline': inline,
			}"
		>
			<div v-if="label" class="label text-uppercase">{{ label }}</div>
			<div v-if="format" class="value fw-bold text-truncate tabular" :class="valueClass">
				<AnimatedNumber :to="number" :format="format" />
			</div>
			<div v-if="sub" class="sub tabular">
				<span v-tooltip="tooltip" :class="{ hint: tooltip?.length }">{{ sub }}</span>
			</div>
			<div v-if="$slots['default']" class="extra"><slot /></div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";

// label, animated number and subline. The default slot renders underneath everything
export default defineComponent({
	name: "Stat",
	components: { AnimatedNumber },
	props: {
		label: String,
		number: { type: Number, default: 0 },
		format: Function as PropType<(n: number) => string>,
		sub: String,
		// text or table rows, see the tooltip directive
		tooltip: [String, Array] as PropType<string | string[][]>,
		valueClass: String,
		// compact and sized by its content: card headers
		inline: Boolean,
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.container-wrapper {
	container-type: inline-size;
}
.stat {
	display: grid;
	/* a chart's inline width from the first layout must not hold the column open */
	grid-template-columns: minmax(0, 1fr);
	grid-template-areas: "label" "value" "sub" "extra";
}
.label {
	grid-area: label;
	color: var(--evcc-gray);
	font-size: 14px;
	margin-bottom: 0.5rem;
	/* wraps between words, a single word too long for the column gets an ellipsis */
	overflow: hidden;
	text-overflow: ellipsis;
}
.value {
	grid-area: value;
	font-size: 28px;
	line-height: 1.2;
	min-width: 0;
}
.sub {
	grid-area: sub;
	color: var(--evcc-gray);
	font-size: 14px;
}
.extra {
	grid-area: extra;
}
.hint {
	text-decoration: underline dotted;
	text-underline-offset: 0.2em;
	cursor: help;
}
/* narrow columns have no room for four digit values at full size */
@container (max-width: 11rem) {
	.value {
		font-size: 24px;
	}
}
.stat--compact .label {
	font-size: 12px;
	margin-bottom: 0.25rem;
}
.stat--compact .value {
	font-size: 20px;
}
/* two of them side by side on a phone, sized by the number: never cut */
.stat--inline .value {
	overflow: visible;
	font-size: 18px;
}
.stat--inline .label,
.stat--inline .sub {
	white-space: nowrap;
}
.stat--inline .sub {
	font-size: 12px;
}
@media (--sm-and-up) {
	.stat--inline .value {
		font-size: 20px;
	}
}
</style>
