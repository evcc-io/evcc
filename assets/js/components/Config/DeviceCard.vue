<template>
	<div
		class="root"
		:class="{
			'round-box': !unconfigured,
			'round-box--error': error,
			'round-box--warning': warning,
			'root--unconfigured': unconfigured,
		}"
	>
		<div class="d-flex align-items-center mb-2">
			<div class="icon me-2">
				<slot name="icon" />
			</div>
			<strong
				class="flex-grow-1 text-nowrap text-truncate"
				data-bs-toggle="tooltip"
				:title="name"
				>{{ title }}</strong
			>
			<DeviceCardEditIcon
				:name="name"
				:editable="editable"
				:noEditButton="noEditButton"
				@edit="$emit('edit')"
			/>
		</div>
		<hr class="my-3 divide" />
		<slot name="tags" />
	</div>
</template>

<script>
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";

export default {
	name: "DeviceCard",
	props: {
		name: String,
		title: String,
		editable: Boolean,
		error: Boolean,
		unconfigured: Boolean,
		warning: Boolean,
		noEditButton: Boolean,
	},
	emits: ["edit"],
	components: { DeviceCardEditIcon },
};
</script>

<style scoped>
.root {
	display: block;
	list-style-type: none;
	border-radius: 1rem;
	padding: 1rem 1.5rem;
	min-height: 8rem;
}
.root--unconfigured {
	background: none;
	border: 1px solid var(--evcc-gray-50);
	transition: border-color var(--evcc-transition-fast) linear;
	order: 1; /* unconfigured tiles at the end of the list */
}
.root--unconfigured:hover {
	border-color: var(--evcc-default-text);
}
.root--unconfigured :deep(.value),
.root--unconfigured :deep(.label) {
	color: var(--evcc-gray) !important;
	font-weight: normal !important;
}
.icon:empty {
	display: none;
}
.divide {
	margin-left: -1.5rem;
	margin-right: -1.5rem;
}
</style>
