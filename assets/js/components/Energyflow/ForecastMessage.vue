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
	color: inherit;
	text-decoration: none;
}
.message {
	text-decoration: underline;
}
</style>
