<template>
	<button
		type="button"
		class="btn btn-sm btn-outline-secondary position-relative border-0 p-2 evcc-gray"
		data-testid="loadpoint-settings-button"
	>
		<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
	</button>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/adjust";
import { defineComponent } from "vue";

export default defineComponent({
	name: "LoadpointSettingsButton",
	props: {
		id: [String, Number],
	},
});
</script>
