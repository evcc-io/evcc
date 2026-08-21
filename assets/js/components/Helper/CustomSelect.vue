<template>
	<label
		class="root position-relative keyboard-focus-ring"
		:class="[
			inline ? 'd-inline-block align-baseline' : 'd-block',
			{ 'focus-ring-visible': showFocusRing },
		]"
		:for="id"
		role="button"
		@mousedown="mouseDown = true"
	>
		<select
			:id="id"
			:value="selected"
			:aria-label="ariaLabel"
			class="custom-select"
			tabindex="0"
			@change="change"
			@focus="handleFocus"
			@keydown="showFocusRing = true"
			@blur="handleBlur"
		>
			<option
				v-for="{ name, value, count, disabled } in options"
				:key="value"
				:value="value"
				:disabled="count === 0 || disabled"
			>
				{{ text(name, count) }}
			</option>
		</select>
		<slot></slot>
	</label>
</template>

<script lang="ts">
import type { SelectOption } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "CustomSelect",
	props: {
		options: { type: Array as PropType<SelectOption<number | string>[]> },
		selected: { type: [String, Number] },
		id: { type: String },
		inline: Boolean,
		ariaLabel: { type: String },
	},
	emits: ["change"],
	data() {
		return {
			// selects match :focus-visible even on mouse interaction, track modality manually
			mouseDown: false,
			showFocusRing: false,
		};
	},
	methods: {
		text(name: string, count?: number) {
			if (count === undefined) {
				return name;
			}
			return `${name} (${count})`;
		},
		change(event: Event) {
			this.$emit("change", event);
		},
		handleFocus() {
			this.showFocusRing = !this.mouseDown;
			this.mouseDown = false;
		},
		handleBlur() {
			this.showFocusRing = false;
			this.mouseDown = false;
		},
	},
});
</script>
<style scoped>
.custom-select {
	left: 0;
	top: 0;
	bottom: 0;
	width: 100%;
	cursor: pointer;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
	/* WebKit propagates the select's intrinsic option width as overflow, causing page scroll */
	contain: paint;
}
</style>
