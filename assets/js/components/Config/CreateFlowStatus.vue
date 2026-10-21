<template>
	<div v-if="inProgress" class="d-flex align-items-start gap-3 py-2">
		<div class="spinner-border flex-shrink-0 text-gray" role="status" aria-hidden="true"></div>
		<p class="mb-0 pt-1 text-gray">{{ creatingText }}</p>
	</div>
	<div v-else class="d-flex flex-column gap-4">
		<div class="d-flex align-items-center gap-3">
			<shopicon-regular-checkmark
				size="m"
				class="text-primary flex-shrink-0"
			></shopicon-regular-checkmark>
			<p class="mb-0">{{ successText }}</p>
		</div>
		<div class="d-flex align-items-center justify-content-between gap-3">
			<slot name="actions" />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import "@h2d2/shopicons/es/regular/checkmark";

// Reusable creating -> created status for device creation flows.
export default defineComponent({
	name: "CreateFlowStatus",
	props: {
		inProgress: Boolean,
		creatingText: String,
		successText: String,
	},
});
</script>
