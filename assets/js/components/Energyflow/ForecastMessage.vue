<template>
	<router-link to="/optimize" class="root" @click.stop>
		<template v-if="!inline">{{ $t("main.energyflow.forecast") }} </template>
		<span class="message"><slot /></span>
	</router-link>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "ForecastMessage",
	props: {
		inline: {
			type: Boolean,
			default: false,
		},
	},
});
</script>

<style scoped>
.root {
	background: linear-gradient(
		90deg,
		var(--evcc-battery) 0%,
		var(--evcc-battery) 15%,
		var(--evcc-self) 30%,
		var(--evcc-battery) 50%,
		var(--evcc-battery) 65%,
		var(--evcc-self) 80%,
		var(--evcc-battery) 100%
	);
	background-size: 200% 100%;
	background-clip: text;
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	animation: forecast-sweep 6s linear infinite;
	text-decoration: none;
}
.message {
	text-decoration: underline;
}
@keyframes forecast-sweep {
	0% {
		background-position: 0% 0;
	}
	100% {
		background-position: -200% 0;
	}
}
</style>
