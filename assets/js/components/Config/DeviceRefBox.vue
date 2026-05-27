<template>
	<label
		class="root d-flex align-items-center justify-content-between"
		:class="[compact ? 'py-0 px-2' : 'py-2 px-3', { invalid: error }]"
		@click="$emit('edit')"
	>
		<div class="flex-grow-1 text-truncate">
			<slot>
				<span>{{ title }}</span>
			</slot>
		</div>
		<DeviceCardEditIcon :editable="true" :danger="error" @edit="$emit('edit')" />
	</label>
</template>

<script lang="ts">
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";

export default {
	name: "DeviceRefBox",
	components: { DeviceCardEditIcon },
	props: {
		title: { type: String, default: "" },
		error: { type: Boolean, default: false },
		compact: { type: Boolean, default: false },
	},
	emits: ["edit"],
};
</script>

<style scoped>
.root {
	border: var(--bs-border-width) solid var(--bs-border-color);
	border-radius: var(--bs-border-radius);
	cursor: pointer;
}
.root.invalid {
	border-color: var(--bs-form-invalid-border-color);
}
</style>
