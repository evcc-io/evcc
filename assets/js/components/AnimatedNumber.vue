<template>
	<number :to="to" :format="format" :duration="duration" />
</template>

<script>
const DURATION = 0.5;

export default {
	name: "AnimatedNumber",
	props: {
		to: { type: Number },
		format: { type: Function },
	},
	data() {
		return { duration: 0 };
	},
	watch: {
		to: function () {
			this.duration = DURATION;
		},
	},
};
</script>
