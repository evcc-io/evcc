<template>
	<template v-for="(node, idx) in nodes" :key="node.name">
		<hr v-if="idx > 0 || depth > 0" />
		<div :style="style">
			<p class="my-2 fw-bold">{{ nodeTitle(node) }}</p>
			<DeviceTags :tags="circuitTags(node)" />
		</div>
		<CircuitTags v-if="node.children?.length" :nodes="node.children" :depth="depth + 1" />
	</template>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceTags from "./DeviceTags.vue";
import type { CircuitNode } from "../../utils/circuits";
import { GRID_CONTROL } from "../../types/evcc";

export default defineComponent({
	name: "CircuitTags",
	components: { DeviceTags },
	props: {
		nodes: { type: Array as PropType<CircuitNode[]>, required: true },
		depth: { type: Number, default: 0 },
	},
	computed: {
		style(): Record<string, string> | undefined {
			return this.depth ? { marginLeft: `${this.depth}rem` } : undefined;
		},
	},
	methods: {
		nodeTitle(node: CircuitNode): string {
			if (node.name === GRID_CONTROL) return this.$t("config.hems.title");
			return node.title || "";
		},
		circuitTags(node: CircuitNode) {
			const result: Record<string, object> = {};
			if (node.dimmed) {
				result["dimmed"] = { value: true };
			}
			if (node.curtailed) {
				result["curtailed"] = { value: true };
			}
			const p = node.power || 0;
			if (node.maxPower) {
				result["powerRange"] = {
					value: [p, node.maxPower],
					warning: node.power && node.power >= node.maxPower,
				};
			} else {
				result["power"] = { value: p, muted: true };
			}
			if (node.maxCurrent) {
				result["currentRange"] = {
					value: [node.current || 0, node.maxCurrent],
					warning: node.current && node.current >= node.maxCurrent,
				};
			}
			return result;
		},
	},
});
</script>
