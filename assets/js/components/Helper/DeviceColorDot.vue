<template>
	<button
		v-if="title"
		type="button"
		class="dot-btn bg-transparent border-0 p-0 d-inline-flex"
		:style="{ '--badge-color': color }"
		:aria-label="title"
		@click.stop="toggle"
	>
		<span class="dot" :class="shapeClass" :style="{ backgroundColor: color }"></span>
		<ColorPickerPopover
			v-model="open"
			:anchor-el="anchor"
			:color="explicit"
			:title="title"
			@update:color="save"
		/>
	</button>
	<span
		v-else
		class="dot d-inline-block"
		:class="shapeClass"
		:style="{ backgroundColor: color }"
	></span>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import ColorPickerPopover from "./ColorPickerPopover.vue";
import api from "../../api";

// color dot of a device, opens the color picker for it. Without a title it is a plain dot
export default defineComponent({
	name: "DeviceColorDot",
	components: { ColorPickerPopover },
	props: {
		title: { type: String, default: "" }, // device the color belongs to
		color: { type: String, default: "" }, // shown, resolved
		explicit: { type: String, default: "" }, // configured, empty when automatic
		line: Boolean, // a short line instead of a circle
	},
	data() {
		return { open: false, anchor: null as HTMLElement | null };
	},
	computed: {
		shapeClass(): string {
			return this.line ? "dot--line" : "rounded-circle";
		},
	},
	methods: {
		toggle() {
			this.anchor = this.$el as HTMLElement;
			this.open = !this.open;
		},
		async save(color: string) {
			try {
				await api.put("devicecolors", { title: this.title, color });
			} catch (e) {
				console.error("set device color failed", e);
			}
		},
	},
});
</script>

<style scoped>
.dot {
	width: 1rem;
	height: 1rem;
	flex-shrink: 0;
}
.dot--line {
	height: 2px;
	border-radius: 1px;
	align-self: center;
}
.dot-btn {
	cursor: pointer;
	border-radius: 50%;
	transition: box-shadow 120ms ease-out;
}
.dot-btn:hover {
	box-shadow: 0 0 0 5px color-mix(in srgb, var(--badge-color) 25%, transparent);
}
.dot-btn:focus-visible {
	outline: 2px solid var(--evcc-default-text);
	outline-offset: 2px;
}
</style>
