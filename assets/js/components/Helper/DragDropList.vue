<template>
	<div ref="container" role="list" :aria-label="$t('config.general.dragList')">
		<slot />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	dragAndDrop,
	tearDown,
	updateConfig,
	parents,
	handleParentDrop,
	handleNodeDrop,
	handleNodePointerdown,
	type ParentConfig,
	type DragState,
} from "@formkit/drag-and-drop";

export default defineComponent({
	name: "DragDropList",
	props: {
		values: { type: Array, required: true },
		transfer: Boolean,
		group: String,
		name: String,
		disabled: Boolean,
		draggableValue: {
			type: Function as PropType<(value: unknown) => boolean>,
			default: () => true,
		},
	},
	emits: ["reorder", "transfer", "hover", "dragstart", "dragend"],
	methods: {
		drop(state: DragState<unknown>) {
			const from = state.initialParent.data.config.name;
			if (!this.disabled && from !== this.name) {
				this.$emit("transfer", {
					value: state.draggedNode.data.value,
					from,
					to: this.name,
				});
			}
		},
	},
	watch: {
		disabled(disabled: boolean) {
			const container = this.$refs["container"] as HTMLElement;
			// updateConfig re-registers with defaults rather than merging its argument.
			updateConfig(container, { ...parents.get(container)?.config, disabled });
		},
	},
	beforeUnmount() {
		tearDown(this.$refs["container"] as HTMLElement);
	},
	mounted() {
		const container = this.$refs["container"] as HTMLElement;
		if (container) {
			const config: Partial<ParentConfig<unknown>> = this.transfer
				? {
						group: this.group,
						name: this.name,
						disabled: this.disabled,
						sortable: false,
						draggable: (child) => child.classList.contains("drag-drop-item"),
						draggableValue: this.draggableValue,
						handleNodeKeydown: () => {},
						handleNodePointerdown: (data, state) => {
							if (data.e.pointerType !== "touch") handleNodePointerdown(data, state);
						},
						// FormKit transfers during hover; keep parent-owned values until a real drop.
						performTransfer: ({ targetParent }) =>
							this.$emit("hover", targetParent.data.config.name),
						handleParentDrop: (data, state) => {
							this.drop(state);
							handleParentDrop(data, state);
						},
						handleNodeDrop: (data, state) => {
							this.drop(state);
							handleNodeDrop(data, state);
						},
						onDragstart: ({ draggedNode }) =>
							this.$emit("dragstart", draggedNode.data.value),
						onDragend: () => this.$emit("dragend"),
					}
				: { dragHandle: ".drag-handle" };
			dragAndDrop({
				parent: container,
				getValues: () => [...this.values],
				setValues: (newOrder: unknown[]) => {
					if (!this.transfer) this.$emit("reorder", newOrder);
				},
				config,
			});
		}
	},
});
</script>

<style>
/* Overwrites Animation for Drag-Elements */
.drag-drop-item[style*="position: fixed"],
.drag-drop-item[style*="z-index: 9999"] {
	transition: none !important;
	animation: none !important;
}
</style>
