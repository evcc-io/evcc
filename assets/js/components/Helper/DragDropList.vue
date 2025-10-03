<template>
	<div ref="container" role="list" :aria-label="$t('config.general.dragList')">
		<slot />
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { dragAndDrop } from "@formkit/drag-and-drop";

export default defineComponent({
	name: "DragDropList",
	props: {
		values: { type: Array, required: true },
	},
	emits: ["reorder"],
	mounted() {
		const container = this.$refs["container"] as HTMLElement;
		if (container) {
			dragAndDrop({
				parent: container,
				getValues: () => this.values,
				setValues: (newOrder: unknown[]) => this.$emit("reorder", newOrder),
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
