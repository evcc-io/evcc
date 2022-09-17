<template>
	<button
		type="button"
		data-bs-toggle="modal"
		:data-bs-target="`#loadpointSettingsModal_${id}`"
		class="btn btn-sm btn-outline-secondary position-relative border-0 p-2 evcc-gray"
	>
		<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
	</button>
</template>

<script>
import "@h2d2/shopicons/es/regular/adjust";

export default {
	name: "LoadpointSettingsButton",
	props: {
		id: [String, Number],
	},
};
</script>
