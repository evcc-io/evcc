<template>
	<GenericModal
		id="confirmWithPasswordModal"
		ref="modal"
		:title="modalTitle"
		data-testid="confirm-with-password-modal"
		@close="$emit('close')"
	>
		<p>TEST</p>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "./GenericModal.vue";

export default defineComponent({
	name: "ConfirmWithPasswordModal",
	components: { GenericModal },

	emits: ["close"],
	props: {
		modalTitle: String,
	},
});
</script>
