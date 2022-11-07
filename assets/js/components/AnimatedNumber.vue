<template>
	<number :to="to" :format="format" :duration="animationDuration" />
</template>

<script>
const DURATION = 0.5;

export default {
	name: "AnimatedNumber",
	props: {
		to: { type: Number },
		format: { type: Function },
		noAnimation: { type: Boolean },
		duration: { type: Number, default: DURATION },
	},
	data() {
		return { activeDuration: 0 };
	},
	computed: {
		animationDuration() {
			return this.noAnimation ? 0 : this.activeDuration;
		},
	},
	watch: {
		to: function () {
			this.activeDuration = this.duration;
		},
	},
};
</script>
