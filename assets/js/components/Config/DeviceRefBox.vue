<template>
	<label
		class="root d-flex align-items-center justify-content-between"
		:class="[
			compact ? 'py-0 px-2' : 'py-2 px-3',
			{ invalid: error },
			disabled ? 'disabled-region evcc-gray' : '',
		]"
		@click="$emit('edit')"
	>
		<div class="flex-grow-1 text-truncate">
			<slot>
				<span>{{ title }}</span>
			</slot>
		</div>
		<button
			v-if="disabled"
			type="button"
			class="btn btn-sm btn-pill me-3"
			:aria-label="$t('config.general.enable')"
			data-testid="device-disabled"
			@click.stop="$emit('enable')"
		>
			{{ $t("config.general.disabled") }}
		</button>
		<DeviceCardEditIcon :editable="true" :danger="error" @edit="$emit('edit')" />
	</label>
</template>

<script lang="ts">
import type { disabled } from "happy-dom/lib/PropertySymbol";
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";

export default {
	name: "DeviceRefBox",
	components: { DeviceCardEditIcon },
	props: {
		title: { type: String, default: "" },
		error: { type: Boolean, default: false },
		compact: { type: Boolean, default: false },
		disabled: { type: Boolean, default: false },
	},
	emits: ["edit", "enable"],
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
.disabled-region {
	background-image: repeating-linear-gradient(
		-45deg,
		transparent 0,
		transparent 10px,
		var(--evcc-gray-25) 10px,
		var(--evcc-gray-25) 20px
	);
}
</style>
