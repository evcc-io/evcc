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
	handleDragstart,
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
			// updateConfig re-registers with defaults rather than merging its argument
			updateConfig(container, { ...parents.get(container)?.config, disabled });
		},
	},
	beforeUnmount() {
		tearDown(this.$refs["container"] as HTMLElement);
	},
	mounted() {
		const container = this.$refs["container"] as HTMLElement;
		if (container) {
			// Safari ignores draggable set during pointerdown (FormKit dragHandle), so filter by handle instead
			let grabbed = false;
			const handleConfig: Partial<ParentConfig<unknown>> = {
				handleNodePointerdown: (data, state) => {
					grabbed = !!(data.e.target as Element).closest(".drag-handle");
					// touch in transfer lists uses tap-to-select instead of synthetic drag
					if (grabbed && !(this.transfer && data.e.pointerType === "touch"))
						handleNodePointerdown(data, state);
				},
				handleDragstart: (data, state) =>
					grabbed ? handleDragstart(data, state) : data.e.preventDefault(),
			};
			const config: Partial<ParentConfig<unknown>> = this.transfer
				? {
						...handleConfig,
						group: this.group,
						name: this.name,
						disabled: this.disabled,
						sortable: false,
						draggable: (child) => child.classList.contains("drag-drop-item"),
						draggableValue: this.draggableValue,
						handleNodeKeydown: () => {},
						// FormKit transfers during hover; keep parent-owned values until a real drop
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
				: handleConfig;
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
