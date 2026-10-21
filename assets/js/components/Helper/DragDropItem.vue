<template>
	<div
		class="drag-drop-item d-flex align-items-center p-2 mb-2 border rounded"
		:class="{
			'drag-drop-item--hidden': !visible,
		}"
		role="listitem"
		:aria-label="$t('config.general.dragItem', { title })"
		tabindex="0"
	>
		<div class="drag-handle" aria-hidden="true">
			<shopicon-regular-menu></shopicon-regular-menu>
		</div>
		<div class="flex-grow-1">
			<slot>
				{{ title }}
			</slot>
		</div>
		<div v-if="$slots['actions']" class="drag-drop-item__actions">
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
	/* larger touch target, vertical margin cancels the item padding. Left edge stays free for scrolling */
	align-self: stretch;
	display: flex;
	align-items: center;
	padding: 0.5rem 0.75rem 0.5rem 0.5rem;
	margin: -0.5rem 0;
}

.drag-handle:active {
	cursor: grabbing;
}

.drag-handle > * {
	pointer-events: none;
}

.drag-drop-item:hover .drag-handle {
	opacity: 1;
}

.drag-drop-item__actions {
	display: flex;
	align-items: center;
}
</style>
