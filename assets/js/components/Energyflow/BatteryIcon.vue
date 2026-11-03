<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<path d="M0-.004h48v48H0v-48z" fill="none" />
		<path fill="currentColor" :d="body" />
		<path
			v-if="limit"
			fill="currentColor"
			:transform="limitTransform"
			d="M8 19q-.425 0-.712-.288T7 18t.288-.712T8 17h8q.425 0 .713.288T17 18t-.288.713T16 19zM11 17V8.825L9.1 10.7q-.275.275-.687.288T7.7 10.7q-.275-.275-.275-.7t.275-.7l3.6-3.6q.15-.15.325-.212T12 5.425t.375.063t.325.212l3.6 3.6q.275.275.288.688T16.3 10.7q-.275.275-.7.275t-.7-.275L13 8.825V17z"
		/>
		<template v-else-if="grid">
			<path
				fill="currentColor"
				transform="translate(24 34.4) scale(0.42) translate(-24 -24)"
				d="m43.109 14.336-12-8a2 2 0 0 0-.151-.08 2 2 0 0 0-.19-.1 2 2 0 0 0-.255-.079c-.04-.011-.077-.027-.118-.035A2 2 0 0 0 29.981 6h-12a2 2 0 0 0-.378.039c-.05.01-.1.029-.145.042a1.4 1.4 0 0 0-.49.211c-.027.016-.055.025-.081.043h-.008L4.891 14.336A2 2 0 0 0 6 18h10v5.528L8.211 39.106a2 2 0 0 0 2.649 2.7L24 35.549l13.14 6.258a2 2 0 0 0 2.649-2.7L32 23.528V18h10a2 2 0 0 0 1.109-3.664M20 10h8v4h-8Zm-7.394 4L16 11.738V14Zm16.158 12 1.151 2.3L24 31.119 18.085 28.3l1.151-2.3Zm-14.375 9.7 1.911-3.819 3.052 1.453Zm14.263-2.362 3.048-1.457 1.911 3.819ZM28 22h-8v-4h8Zm4-8v-2.263L35.4 14Z"
			/>
			<path
				fill="currentColor"
				:transform="arrowTransform"
				d="M11.288 13.712Q11 13.425 11 13V8.825L9.1 10.7q-.275.275-.687.288T7.7 10.7q-.275-.275-.275-.7t.275-.7l3.6-3.6q.15-.15.325-.212T12 5.425t.375.063t.325.212l3.6 3.6q.275.275.288.688T16.3 10.7q-.275.275-.7.275t-.7-.275L13 8.825V13q0 .425-.287.713T12 14t-.712-.288"
			/>
		</template>
		<path v-else fill="currentColor" :d="socRect" />
	</svg>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import icon from "@/mixins/icon";
import { BATTERY_MODE } from "@/types/evcc";

const CLOSED_BODY =
	"M35 9.996h-3v-4a2 2 0 00-2-2H18a2 2 0 00-2 2v4h-3a2 2 0 00-2 2v30a2 2 0 002 2h22a2 2 0 002-2v-30a2 2 0 00-2-2zm-15-2h8v2h-8v-2zm13 32H15v-26h18v26z";
// bottom edge cut away so the grid pylon can stand on the open legs
const OPEN_BODY =
	"M35 9.996h-3v-4a2 2 0 00-2-2H18a2 2 0 00-2 2v4h-3a2 2 0 00-2 2v28a2 2 0 002 2h2v-28h18v28h2a2 2 0 002-2v-28a2 2 0 00-2-2zm-15-2h8v2h-8v-2z";

export default defineComponent({
	name: "BatteryIcon",
	mixins: [icon],
	props: {
		soc: { type: Number, default: 0 },
		mode: { type: String as PropType<BATTERY_MODE>, default: BATTERY_MODE.NORMAL },
	},
	computed: {
		limit() {
			return [BATTERY_MODE.HOLD, BATTERY_MODE.HOLDCHARGE].includes(this.mode);
		},
		grid() {
			return [BATTERY_MODE.CHARGE, BATTERY_MODE.DISCHARGE].includes(this.mode);
		},
		body() {
			return this.grid ? OPEN_BODY : CLOSED_BODY;
		},
		flipped() {
			return [BATTERY_MODE.HOLDCHARGE, BATTERY_MODE.DISCHARGE].includes(this.mode);
		},
		limitTransform() {
			const flip = this.flipped ? " rotate(180 12 12)" : "";
			return `translate(24 27) scale(1.3) translate(-12 -12)${flip}`;
		},
		arrowTransform() {
			const flip = this.flipped ? " rotate(180)" : "";
			return `translate(24 20.5)${flip} translate(-12 -9.7)`;
		},
		socRect() {
			const height = (this.soc / (100 / 22)).toFixed(2);
			return `M30 38H18v-${height}h12v${height}z`;
		},
	},
});
</script>
