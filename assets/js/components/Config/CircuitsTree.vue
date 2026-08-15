<template>
	<div>
		<div class="d-flex align-items-stretch row-spacing">
			<template v-if="depth > 0">
				<span v-for="(full, i) in guides" :key="i" class="tree-col">
					<span v-if="full" class="tree-line" />
				</span>
				<span class="tree-col">
					<span
						class="tree-line"
						:class="{ 'tree-line--half': isLast }"
					/>
					<span class="tree-knick" />
				</span>
			</template>
			<DeviceRefBox
				compact
				class="flex-grow-1"
				@edit="$emit('edit', circuitsTree)"
			>
				<span class="d-flex align-items-center gap-2">
					<span class="fw-bold">{{ circuitsTree?.title }}</span>
					<span class="ms-auto evcc-gray value">{{
						valueLabel
					}}</span>
				</span>
			</DeviceRefBox>
		</div>

		<CircuitsTree
			v-for="child in circuitsTree?.circuitChilds"
			:key="child.title"
			:circuits-tree="child"
			:depth="depth + 1"
			:guides="childGuides"
			:is-last="false"
			@edit="$emit('edit', $event)"
		/>

		<div class="d-flex align-items-stretch row-spacing">
			<span v-for="(full, i) in childGuides" :key="i" class="tree-col">
				<span v-if="full" class="tree-line" />
			</span>
			<span class="tree-col">
				<span class="tree-line tree-line--half" />
				<span class="tree-knick" />
			</span>
			<span class="d-flex align-items-center gap-2 evcc-gray add-link">
				<shopicon-regular-plus size="xs"></shopicon-regular-plus>
				<span>Add sub-circuit</span>
			</span>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import type { RecursiveCircuit } from "./CircuitsModal.vue";
import DeviceRefBox from "./DeviceRefBox.vue";
import formatter from "@/mixins/formatter.ts";

export default {
	name: "CircuitsTree",
	mixins: [formatter],
	components: { DeviceRefBox },
	props: {
		circuitsTree: {
			type: Object as PropType<RecursiveCircuit>,
		},
		/** Nesting depth from root (0 = root, no indentation/lines). */
		depth: { type: Number, default: 0 },
		/**
		 * One bool per ancestor level (excluding the direct parent): true means
		 * the vertical line continues through that column (the ancestor has
		 * more siblings after it), false means the column stays empty (the
		 * ancestor was itself the last child).
		 */
		guides: { type: Array as PropType<boolean[]>, default: () => [] },
		isLast: { type: Boolean, default: false },
	},
	emits: ["edit"],
	computed: {
		childGuides(): boolean[] {
			return this.depth === 0 ? [] : [...this.guides, !this.isLast];
		},
		valueLabel(): string {
			if (!this.circuitsTree) return "";
			const { power, current } = this.circuitsTree;
			const parts: string[] = [];
			if (power !== undefined) parts.push(this.fmtW(power));
			if (current !== undefined) parts.push(`${current} A`);
			return parts.join(" · ");
		},
	},
};
</script>

<style scoped>
.row-spacing {
	margin-bottom: 4px;
}
.tree-col {
	width: 22px;
	position: relative;
	flex: 0 0 auto;
}
.tree-line {
	position: absolute;
	left: 10px;
	top: 0;
	bottom: -4px;
	width: 1px;
	background: var(--evcc-gray-25);
}
.tree-line--half {
	top: 0;
	bottom: auto;
	height: 50%;
}
.tree-knick {
	position: absolute;
	left: 10px;
	top: 50%;
	width: 12px;
	height: 1px;
	background: var(--evcc-gray-25);
}
.value {
	font-size: 12.5px;
	font-variant-numeric: tabular-nums;
}
.add-link {
	font-size: 13.5px;
	padding: 6px 0;
}
</style>
