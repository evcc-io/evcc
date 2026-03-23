<template>
	<div
		class="alert alert-danger d-flex justify-content-between"
		data-testid="invalid-reference-alert"
	>
		<span>
			{{ message }}: <strong>{{ value }}</strong>
		</span>
		<a href="#" class="text-danger ms-3" @click.prevent="$emit('remove')">
			{{ $t("config.general.remove") }}
		</a>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "InvalidReferenceAlert",
	props: {
		message: { type: String, required: true },
		value: { type: String, default: "" },
	},
	emits: ["remove"],
});
</script>
