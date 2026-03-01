<template>
	<router-link to="/optimize" class="root" @click.stop>
		{{ $t("main.energyflow.forecast") }} <span class="message">{{ message }}</span>
	</router-link>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "ForecastMessage",
	props: {
		message: { type: String, required: true },
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
