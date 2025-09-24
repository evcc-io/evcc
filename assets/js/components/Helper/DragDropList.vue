<template>
	<div ref="container">
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
				setValues: (newOrder) => this.$emit("reorder", newOrder),
			});
		}
	},
});
</script>
