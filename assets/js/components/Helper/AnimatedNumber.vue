<template>
	<span />
</template>

<script lang="ts">
import { CountUp } from "countup.js";
import { defineComponent, type PropType } from "vue";
const DURATION = 0.5;

export default defineComponent({
	name: "AnimatedNumber",
	props: {
		to: { type: Number, default: 0 },
		format: { type: Function as PropType<(n: number) => string>, required: true },
		duration: { type: Number, default: DURATION },
	},
	data() {
		return {
			instance: null as CountUp | null,
		};
	},
	watch: {
		to(value) {
			this.instance?.update(value);
		},
	},
	mounted() {
		if (this.instance) {
			return;
		}
		this.instance = new CountUp(this.$el, this.to, {
			startVal: this.to,
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
	},
	methods: {
		forceUpdate() {
			this.instance?.reset();
			this.instance?.update(this.to);
		},
	},
});
</script>
