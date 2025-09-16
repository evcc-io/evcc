<template>
	<div
		class="drag-drop-item d-flex align-items-center p-2 mb-2 border rounded"
		:class="{
			'drag-drop-item--hidden': !visible,
			'drag-drop-item--dragging': isDragging,
		}"
		draggable="true"
		@dragstart="$emit('dragstart', $event)"
		@dragover.prevent
		@drop="$emit('drop', $event)"
		@touchstart="$emit('touchstart', $event)"
		@touchmove="$emit('touchmove', $event)"
		@touchend="$emit('touchend', $event)"
	>
		<div class="drag-handle me-2">
			<shopicon-regular-menu></shopicon-regular-menu>
		</div>
		<div class="flex-grow-1">
			<slot name="content">
				{{ title }}
			</slot>
		</div>
		<div v-if="$slots['actions']" class="drag-drop-item__actions">
			<slot name="actions"></slot>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/menu";
import { defineComponent } from "vue";

export default defineComponent({
	name: "DragDropItem",
	props: {
		/**
		 * Display title for the item
		 */
		title: {
			type: String,
			default: "",
		},
		/**
		 * Whether the item is visible/enabled
		 */
		visible: {
			type: Boolean,
			default: true,
		},
		/**
		 * Whether the item is currently being dragged
		 */
		isDragging: {
			type: Boolean,
			default: false,
		},
	},
	emits: ["dragstart", "drop", "touchstart", "touchmove", "touchend"],
});
</script>

<style scoped>
.drag-drop-item {
	cursor: move;
	user-select: none;
	background-color: var(--evcc-box);
	border-color: var(--bs-border-color-translucent) !important;
	transition: all var(--evcc-transition-fast) ease-in-out;
}

.drag-drop-item--hidden {
	opacity: 0.6;
}

.drag-drop-item--dragging {
	opacity: 0.8;
	transform: scale(1.02) rotate(0deg);
	box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
	z-index: 10;
}

.drag-handle {
	color: var(--bs-secondary);
	opacity: 0.7;
	transition: opacity var(--evcc-transition-fast) ease-in-out;
	cursor: grab;
}

.drag-handle:active {
	cursor: grabbing;
}

.drag-drop-item:hover .drag-handle {
	opacity: 1;
}

.drag-drop-item__actions {
	display: flex;
	align-items: center;
}
</style>
