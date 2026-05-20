<template>
	<ul
		class="root p-0 d-flex flex-wrap column-gap-4 row-gap-2"
		:class="{
			'root--small-equal-widths': smallEqualWidths,
			'root--grid': grid,
		}"
	>
		<li
			v-for="legend in legends"
			:key="legend.label"
			class="legend-item d-flex align-items-baseline gap-2 no-wrap"
			:class="{
				'legend-item--focusable': legend.focusable,
				'legend-item--dim': legend.dim,
			}"
			:role="legend.focusable ? 'button' : undefined"
			:tabindex="legend.focusable ? 0 : undefined"
			@click="legend.focusable && onFocus(legend)"
			@keydown.enter.prevent="legend.focusable && onFocus(legend)"
			@keydown.space.prevent="legend.focusable && onFocus(legend)"
		>
			<button
				v-if="legend.color && isPickable(legend)"
				type="button"
				class="legend-color-btn align-self-center me-1 bg-transparent border-0 p-0 d-inline-flex"
				:style="{ '--badge-color': legend.color }"
				:aria-label="legend.label"
				@click.stop="onPick(legend, $event)"
			>
				<span
					class="legend-color"
					:class="colorClass(legend)"
					:style="{
						backgroundColor: legend.color,
						borderColor: legend.color,
					}"
				></span>
			</button>
			<span
				v-else-if="legend.color"
				class="legend-color align-self-center me-1"
				:class="colorClass(legend)"
				:style="{
					backgroundColor: legend.color,
					borderColor: legend.color,
				}"
			></span>
			<div class="legend-label text-nowrap">{{ legend.label }}</div>
			<div
				v-for="value in valueList(legend.value)"
				:key="value"
				class="text-muted text-nowrap legend-value text-end"
			>
				{{ value }}
			</div>
		</li>
		<ColorPickerPopover
			v-model="pickerOpen"
			:anchor-el="pickerAnchor"
			:color="pickerColor"
			:title="pickerTitle"
			@update:color="onColorChange"
		/>
	</ul>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { Legend } from "./types";
import type { DeviceColors } from "@/types/evcc";
import ColorPickerPopover from "../Helper/ColorPickerPopover.vue";
import api from "../../api";

export default defineComponent({
	name: "LegendList",
	components: { ColorPickerPopover },
	props: {
		legends: Array as PropType<Legend[]>,
		grid: Boolean,
		smallEqualWidths: Boolean,
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	emits: ["focus"],
	data() {
		return {
			pickerOpen: false,
			pickerAnchor: null as HTMLElement | null,
			pickerId: "",
		};
	},
	computed: {
		pickerTitle(): string {
			const found = (this.legends || []).find((l) => l.id === this.pickerId);
			return found?.label || this.pickerId;
		},
		pickerColor(): string {
			return this.deviceColors[this.pickerId] || "";
		},
	},
	methods: {
		valueList(value: Legend["value"]) {
			if (!value) return [];
			return Array.isArray(value) ? value : [value];
		},
		colorClass(legend: Legend) {
			return legend.type === "line" ? "legend-color--line" : "legend-color--area";
		},
		isPickable(legend: Legend) {
			return !!legend.id;
		},
		onPick(legend: Legend, e: MouseEvent) {
			if (!legend.id) return;
			if (this.pickerOpen && this.pickerId === legend.id) {
				this.pickerOpen = false;
				return;
			}
			this.pickerAnchor = (e.currentTarget as HTMLElement) || null;
			this.pickerId = legend.id;
			this.pickerOpen = true;
		},
		onFocus(legend: Legend) {
			this.$emit("focus", legend);
		},
		async onColorChange(color: string) {
			if (!this.pickerId) return;
			try {
				await api.put("devicecolors", { title: this.pickerId, color });
			} catch (e) {
				console.error("set device color failed", e);
			}
		},
	},
});
</script>

<style scoped>
.root {
	justify-content: flex-start;
}
.legend-color {
	width: 1rem;
	height: 1rem;
	flex-shrink: 0;
	display: inline-block;
}
.legend-item--focusable {
	cursor: pointer;
	user-select: none;
	transition: opacity 150ms;
}
.legend-item--dim {
	opacity: 0.35;
}

.legend-color--area {
	border-radius: 50%;
}

.legend-color--line {
	height: 2px;
	border-radius: 1px;
	align-self: center;
}

.legend-color-btn {
	cursor: pointer;
	border-radius: 50%;
	transition: box-shadow 120ms ease-out;
}
.legend-color-btn:hover {
	box-shadow: 0 0 0 5px color-mix(in srgb, var(--badge-color) 25%, transparent);
}
.legend-color-btn:focus-visible {
	outline: 2px solid var(--evcc-default-text);
	outline-offset: 2px;
}

.legend-label {
	flex-shrink: 0;
	flex-grow: 0;
}

.root--grid .legend-label {
	flex-grow: 1;
	flex-shrink: 1;
	text-overflow: ellipsis;
	overflow: hidden;
}
.root--grid .legend-item {
	flex-grow: 1;
	flex-basis: 100%;
}
.root--grid .legend-value:last-child {
	flex-basis: 3.5rem;
}

.root--small-equal-widths {
	display: flex;
	justify-content: space-evenly;
}
.root--small-equal-widths .legend-item {
	flex-basis: 8rem;
}
.root--small-equal-widths .legend-label {
	flex-grow: 1;
	flex-shrink: 1;
}
</style>
