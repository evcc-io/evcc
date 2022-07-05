<template>
	<number :to="to" :format="format" :duration="duration" />
</template>

<script>
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
			this.duration = 1;
		},
	},
};
</script>
