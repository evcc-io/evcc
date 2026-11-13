<template>
	<div class="position-relative d-inline-flex flex-shrink-0">
		<svg viewBox="0 0 40 40" :width="size" :height="size" aria-hidden="true">
			<!-- the gaps are cut by lines of even width, a dash gap would open like a wedge -->
			<mask :id="maskId" maskUnits="userSpaceOnUse">
				<rect width="40" height="40" fill="white" />
				<line
					v-for="arc in arcs"
					:key="arc.key"
					x1="20"
					y1="20"
					x2="20"
					y2="0"
					stroke="black"
					:stroke-width="arc.cut"
					:style="{ transform: `rotate(${arc.angle}deg)` }"
				/>
			</mask>
			<g :mask="`url(#${maskId})`">
				<circle
					v-for="arc in arcs"
					:key="arc.key"
					cx="20"
					cy="20"
					:r="radius"
					fill="none"
					:stroke="arc.color"
					:stroke-width="stroke"
					pathLength="100"
					:style="{ strokeDasharray: arc.dash, strokeDashoffset: arc.offset }"
					transform="rotate(-90 20 20)"
				/>
			</g>
		</svg>
		<div
			v-if="$slots['default']"
			class="position-absolute top-50 start-50 translate-middle text-center"
		>
			<slot />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

const BOX = 40;
const GAP = 2; // px
let count = 0; // a mask id per ring

// shares of a whole as a ring: arcs proportional to their values, starting at the
// top, with a hairline gap of even width between them
export default defineComponent({
	name: "ShareRing",
	props: {
		segments: { type: Array as PropType<{ value: number; color: string }[]>, required: true },
		size: { type: Number, default: 28 }, // px
		stroke: { type: Number, default: 9 }, // of the 40 unit box
	},
	data() {
		return { maskId: `share-ring-${count++}` };
	},
	computed: {
		radius(): number {
			return (BOX - this.stroke) / 2 - 0.5;
		},
		arcs(): {
			key: number;
			color: string;
			dash: string;
			offset: number;
			angle: number;
			cut: number;
		}[] {
			// empty segments stay as zero length arcs, so they grow and shrink in place
			const live = this.segments.filter((s) => s.value > 0);
			const total = live.reduce((acc, s) => acc + s.value, 0);
			// a cut where each arc starts, none for a single arc
			const cut = live.length > 1 ? (GAP * BOX) / this.size : 0;
			let offset = 0;
			return this.segments.map((s, key) => {
				const length = s.value > 0 ? (s.value / total) * 100 : 0;
				const arc = {
					key,
					color: s.color,
					dash: `${length} ${100 - length}`,
					offset: -offset,
					angle: offset * 3.6,
					cut: length ? cut : 0,
				};
				offset += length;
				return arc;
			});
		},
	},
});
</script>

<style scoped>
circle {
	transition:
		stroke-dasharray var(--evcc-transition-medium) ease,
		stroke-dashoffset var(--evcc-transition-medium) ease;
}
line {
	transform-origin: 20px 20px;
	transition: transform var(--evcc-transition-medium) ease;
}
@media (prefers-reduced-motion: reduce) {
	circle,
	line {
		transition: none;
	}
}
</style>
