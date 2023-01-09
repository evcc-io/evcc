<template>
	<button
		type="button"
		class="btn btn-sm btn-outline-secondary position-relative border-0 p-2 evcc-gray"
		@click="openModal"
	>
		<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
	</button>
</template>

<script>
import "@h2d2/shopicons/es/regular/adjust";
import Modal from "bootstrap/js/dist/modal";

export default {
	name: "LoadpointSettingsButton",
	props: {
		id: [String, Number],
	},
	methods: {
		openModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById(`loadpointSettingsModal_${this.id}`)
			);
			modal.show();
		},
	},
};
</script>
