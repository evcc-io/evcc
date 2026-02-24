<template>
	<div
		class="root"
		:class="{
			'round-box': !unconfigured,
			'round-box--error': error,
			'round-box--warning': warning,
			'root--unconfigured': unconfigured,
			'root--with-tags': $slots.tags,
		}"
	>
		<div class="d-flex align-items-center" :class="{ 'mb-2': $slots.tags }">
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
				:editable="editable"
				:noEditButton="noEditButton"
				:badge="badge"
				@edit="$emit('edit')"
			/>
		</div>
		<div v-if="$slots.tags">
			<hr class="my-3 divide" />
			<slot name="tags" />
		</div>
	</div>
</template>

<script>
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";

export default {
	name: "DeviceCard",
	components: { DeviceCardEditIcon },
	props: {
		name: String,
		title: String,
		editable: Boolean,
		error: Boolean,
		unconfigured: Boolean,
		warning: Boolean,
		noEditButton: Boolean,
		badge: Boolean,
	},
	emits: ["edit"],
};
</script>

<style scoped>
.root {
	display: block;
	list-style-type: none;
	border-radius: 1rem;
	padding: 1rem 1.5rem;
}
.root--with-tags {
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
