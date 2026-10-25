<template>
	<div
		class="drag-drop-item d-flex mb-2 border rounded"
		:class="{
			'drag-drop-item--hidden': !visible,
		}"
		role="listitem"
		:aria-label="$t('config.general.dragItem', { title })"
		tabindex="0"
	>
		<div
			class="drag-handle d-flex align-items-center px-2"
			aria-hidden="true"
			data-testid="drag-handle"
		>
			<shopicon-regular-menu></shopicon-regular-menu>
		</div>
		<div class="flex-grow-1 d-flex align-items-center py-2">
			<slot>
				{{ title }}
			</slot>
		</div>
		<div v-if="$slots['actions']" class="d-flex align-items-center py-2 pe-2">
			<slot name="actions"></slot>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import "@h2d2/shopicons/es/regular/menu";

export default defineComponent({
	name: "DragDropItem",
	props: {
		title: { type: String, default: "" },
		visible: { type: Boolean, default: true },
	},
});
</script>

<style scoped>
.drag-drop-item {
	user-select: none;
	background-color: var(--evcc-box);
	border-color: var(--bs-border-color-translucent) !important;
	transition: all var(--evcc-transition-fast) ease-in-out;
	transform: translate(0, 0);
}

.drag-drop-item--hidden {
	opacity: 0.6;
}

.drag-handle {
	color: var(--bs-secondary);
	opacity: 0.7;
	transition: opacity var(--evcc-transition-fast) ease-in-out;
	cursor: grab;
	touch-action: none;
}

.drag-handle:active {
	cursor: grabbing;
}

.drag-handle > * {
	pointer-events: none;
}

.drag-handle:hover {
	opacity: 1;
}
</style>
