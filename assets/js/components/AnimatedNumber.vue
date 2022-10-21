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
	},
	data() {
		return { duration: 0 };
	},
	computed: {
		animationDuration() {
			return this.noAnimation ? 0 : this.duration;
		},
	},
	watch: {
		to: function () {
			this.duration = DURATION;
		},
	},
};
</script>
