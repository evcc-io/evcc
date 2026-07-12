<template>
	<div v-if="inProgress" class="d-flex align-items-start gap-3 py-2">
		<div class="spinner-border flex-shrink-0 text-gray" role="status" aria-hidden="true"></div>
		<p class="mb-0 pt-1 text-gray">{{ creatingText }}</p>
	</div>
	<div v-else class="d-flex flex-column gap-4">
		<div class="d-flex align-items-start gap-3">
			<SuccessCheck class="flex-shrink-0" />
			<p class="mb-0 pt-2">{{ successText }}</p>
		</div>
		<div class="d-flex align-items-center justify-content-between gap-3">
			<slot name="actions" />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import SuccessCheck from "../Helper/SuccessCheck.vue";

// Reusable creating -> created status for device creation flows.
export default defineComponent({
	name: "CreateFlowStatus",
	components: { SuccessCheck },
	props: {
		inProgress: Boolean,
		creatingText: String,
		successText: String,
	},
});
</script>
