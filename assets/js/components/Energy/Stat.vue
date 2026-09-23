<template>
	<!-- the wrapper is the query container, rules cannot address the container itself -->
	<div class="container-wrapper">
		<div class="stat" :class="{ 'stat--compact': compact, 'stat--aside': $slots['aside'] }">
			<div class="label text-uppercase" :class="alignClass">{{ label }}</div>
			<div
				v-if="format"
				class="value fw-bold text-truncate"
				:class="[valueClass, alignClass]"
			>
				<AnimatedNumber :to="number" :format="format" />
			</div>
			<div v-if="sub || $slots['sub']" class="sub" :class="alignClass">
				<slot name="sub">
					<span v-tooltip="tooltip" :class="{ hint: tooltip }">{{ sub }}</span>
				</slot>
			</div>
			<div v-if="$slots['aside']" class="aside"><slot name="aside" /></div>
			<div v-if="$slots['asideSub']" class="aside-sub small text-muted">
				<slot name="asideSub" />
			</div>
			<div v-if="$slots['default']" class="extra"><slot /></div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";

// label, animated number and subline. `aside` sits beside the number when the
// component is wide enough, below it otherwise, `asideSub` follows it. The default
// slot renders underneath everything
export default defineComponent({
	name: "Stat",
	components: { AnimatedNumber },
	props: {
		label: String,
		number: { type: Number, default: 0 },
		format: Function as PropType<(n: number) => string>,
		sub: String,
		tooltip: String,
		valueClass: String,
		compact: Boolean,
		// Bootstrap alignment suffixes, several for responsive changes: "center lg-start"
		align: { type: String, default: "start" },
	},
	computed: {
		alignClass(): string[] {
			return this.align.split(" ").map((a) => `text-${a}`);
		},
	},
});
</script>

<style scoped>
.container-wrapper {
	container-type: inline-size;
}
.stat {
	display: grid;
	/* a chart's inline width from the first layout must not hold the column open */
	grid-template-columns: minmax(0, 1fr);
	grid-template-areas: "label" "value" "sub" "aside" "aside-sub" "extra";
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
.aside {
	grid-area: aside;
	min-width: 0;
	margin-top: 1rem;
}
.aside-sub {
	grid-area: aside-sub;
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
/* enough room: aside shares the number's row, more of it on wide cards */
@container (min-width: 24rem) {
	.stat--aside {
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		column-gap: 1.5rem;
		grid-template-areas: "label ." "value aside" "sub aside-sub" "extra extra";
	}
	.stat--aside .aside {
		margin-top: 0;
	}
	.stat--aside .aside-sub {
		text-align: end;
	}
}
@container (min-width: 32rem) {
	.stat--aside {
		grid-template-columns: minmax(0, 1fr) minmax(0, 2fr);
	}
}
</style>
