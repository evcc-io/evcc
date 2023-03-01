<template>
	<span />
</template>

<script>
import { CountUp } from "countup.js";
const DURATION = 0.5;

export default {
	name: "AnimatedNumber",
	props: {
		to: { type: Number, default: 0 },
		format: { type: Function, required: true },
		duration: { type: Number, default: DURATION },
	},
	data() {
		return {
			instance: null,
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
			decimalPlaces: 2,
		});
		if (this.instance.error) {
			console.error(this.instance.error);
		}
	},
	unmounted() {
		this.instance = null;
	},
};
</script>

<style lang="less" scoped></style>
