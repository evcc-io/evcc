<template>
	<span />
</template>

<script lang="ts">
import { CountUp } from "countup.js";
import { defineComponent, type PropType } from "vue";
import type { Timeout } from "@/types/evcc";
const DURATION = 0.5;

export default defineComponent({
	name: "AnimatedNumber",
	props: {
		to: { type: [String, Number], default: 0 },
		format: { type: Function as PropType<(n: number) => string>, required: true },
		duration: { type: Number, default: DURATION },
	},
	data() {
		return {
			instance: null as CountUp | null,
			timeout: null as Timeout | null,
		};
	},
	watch: {
		to(value) {
			this.update(value);
		},
	},
	mounted() {
		if (this.instance) {
			return;
		}
		this.instance = new CountUp(this.$el, Number(this.to), {
			startVal: Number(this.to),
			formattingFn: this.format,
			duration: this.duration,
			decimalPlaces: 3,
		});
		if (this.instance.error) {
			console.error(this.instance.error);
		}
	},
	unmounted() {
		this.instance = null;
		if (this.timeout !== null) {
			clearTimeout(this.timeout);
		}
	},
	methods: {
		forceUpdate() {
			this.instance?.reset();
			this.update(Number(this.to));
		},
		update(value: number) {
			// debounced to avoid rendering issues
			// @see https://github.com/inorganik/countUp.js/issues/330#issuecomment-2697595198
			if (this.timeout !== null) {
				clearTimeout(this.timeout);
			}
			this.timeout = setTimeout(() => {
				this.instance?.update(value);
			}, 100);
		},
	},
});
</script>
