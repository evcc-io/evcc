<template>
	<div
		class="drag-drop-item d-flex align-items-center p-2 mb-2 border rounded"
		:class="{
			'drag-drop-item--hidden': !visible,
			'drag-drop-item--assigned': assigned,
			'drag-drop-item--selected': selected,
			'opacity-50': disabled,
		}"
		role="listitem"
		:aria-label="$t('config.general.dragItem', { title })"
		tabindex="0"
		:aria-selected="interactive ? selected : undefined"
		:aria-disabled="disabled || undefined"
		@click.stop="activate"
		@keydown.enter.self="activate"
		@keydown.space.self="activate"
	>
		<div class="drag-handle me-2" aria-hidden="true">
			<shopicon-regular-menu></shopicon-regular-menu>
		</div>
		<div class="flex-grow-1 min-w-0 text-truncate">
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
		interactive: Boolean,
		selected: Boolean,
		disabled: Boolean,
		assigned: Boolean,
	},
	emits: ["activate"],
	methods: {
		activate(event: Event) {
			if (!this.interactive || this.disabled) return;
			event.preventDefault();
			this.$emit("activate");
		},
	},
});
</script>

<style scoped>
.drag-drop-item {
	cursor: grab;
	user-select: none;
	background-color: var(--evcc-box);
	border-color: var(--bs-border-color-translucent) !important;
	transition: all var(--evcc-transition-fast) ease-in-out;
	transform: translate(0, 0);
}

.drag-drop-item:active {
	cursor: grabbing;
}

.drag-drop-item--assigned {
	background: var(--evcc-background);
	border-color: transparent !important;
}
.drag-drop-item--selected {
	box-shadow: 0 0 0 2px var(--bs-primary);
}

.drag-drop-item--hidden {
	opacity: 0.6;
}

.drag-handle {
	color: var(--bs-secondary);
	opacity: 0.7;
	transition: opacity var(--evcc-transition-fast) ease-in-out;
	cursor: inherit;
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
