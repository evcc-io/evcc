<template>
	<div class="mb-3 row">
		<label :for="id" class="col-sm-4 col-form-label pt-0 pt-sm-2">
			{{ label }}
		</label>
		<div class="col-sm-8">
			<div class="row">
				<div class="col-lg-6">
					<slot />
				</div>
			</div>
			<small v-if="description" class="d-block text-muted mt-1">
				{{ description }}
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "SettingsFormRow",
	props: {
		id: String,
		label: String,
		description: String,
	},
});
</script>
