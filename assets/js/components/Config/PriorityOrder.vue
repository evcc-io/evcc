<template>
	<div class="py-2">
		<div class="d-flex align-items-center justify-content-between mb-2">
			<div class="small evcc-gray d-flex align-items-center gap-1" aria-hidden="true">
				<shopicon-regular-arrowup size="s"></shopicon-regular-arrowup>
				<span>{{ $t("config.priority.order.high") }}</span>
			</div>
			<button
				type="button"
				class="btn btn-link btn-sm text-gray p-0 border-0 d-flex align-items-center gap-1 keyboard-focus-ring"
				data-testid="priority-add-tier"
				@click="displayMax++"
			>
				<shopicon-regular-plus size="s"></shopicon-regular-plus>
				<span>{{ $t("config.priority.order.addTier") }}</span>
			</button>
		</div>

		<div
			v-for="lane in lanes"
			:key="lane"
			:ref="setLaneRef(lane)"
			class="lane d-flex align-items-baseline gap-2 mb-2 p-2 pb-0 rounded"
		>
			<span class="lane-num fw-bold text-center" aria-hidden="true">{{ lane }}</span>
			<div
				class="chips d-flex flex-wrap column-gap-2 flex-grow-1"
				role="list"
				:aria-label="$t('config.priority.order.tier', { tier: lane })"
			>
				<DragDropItem
					v-for="name in members(lane)"
					:key="name"
					:title="labels[name] || name"
					@keydown.up.prevent="move(name, 1)"
					@keydown.down.prevent="move(name, -1)"
				/>
				<!-- invisible chip keeps empty lanes at chip height -->
				<DragDropItem
					v-if="!members(lane).length"
					class="invisible"
					title="-"
					tabindex="-1"
					aria-hidden="true"
				/>
			</div>
		</div>

		<div class="small evcc-gray d-flex align-items-center gap-1" aria-hidden="true">
			<shopicon-regular-arrowdown size="s"></shopicon-regular-arrowdown>
			<span>{{ $t("config.priority.order.low") }}</span>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { dragAndDrop } from "@formkit/drag-and-drop";
import DragDropItem from "../Helper/DragDropItem.vue";
import "@h2d2/shopicons/es/regular/arrowup";
import "@h2d2/shopicons/es/regular/arrowdown";
import "@h2d2/shopicons/es/regular/plus";

export default defineComponent({
	name: "PriorityOrder",
	components: { DragDropItem },
	props: {
		modelValue: {
			type: Object as PropType<Record<string, number>>,
			required: true,
		},
		labels: {
			type: Object as PropType<Record<string, string>>,
			default: () => ({}),
		},
	},
	emits: ["update:modelValue"],
	data() {
		return {
			displayMax: 0,
			laneRefs: {} as Record<number, HTMLElement>,
		};
	},
	computed: {
		max(): number {
			return Math.max(0, ...Object.values(this.modelValue));
		},
		lanes(): number[] {
			const res = [];
			for (let i = this.displayMax; i >= 0; i--) res.push(i);
			return res;
		},
	},
	watch: {
		displayMax() {
			this.$nextTick(this.register);
		},
	},
	mounted() {
		// one empty tier above the highest used one; grows only via the add-tier button
		this.displayMax = this.max + 1;
		this.$nextTick(this.register);
	},
	methods: {
		members(lane: number): string[] {
			return Object.keys(this.modelValue).filter((name) => this.modelValue[name] === lane);
		},
		setLaneRef(lane: number) {
			return (el: unknown) => {
				if (el) this.laneRefs[lane] = el as HTMLElement;
			};
		},
		setTier(name: string, value: number) {
			const clamped = Math.min(Math.max(0, value), this.displayMax);
			if (this.modelValue[name] === clamped) return;
			this.$emit("update:modelValue", { ...this.modelValue, [name]: clamped });
		},
		move(name: string, delta: number) {
			// keyboard: one tier per keypress, within the visible lanes
			this.setTier(name, this.modelValue[name] + delta);
		},
		register() {
			for (const lane of this.lanes) {
				const parent = this.laneRefs[lane]?.querySelector(".chips") as HTMLElement;
				if (!parent) continue;
				dragAndDrop({
					parent,
					getValues: () => this.members(lane),
					setValues: (values: unknown[]) => {
						for (const name of values as string[]) this.setTier(name, lane);
					},
					config: {
						group: "priorityOrder",
						draggable: (el: HTMLElement) =>
							el.classList.contains("drag-drop-item") &&
							!el.classList.contains("invisible"),
					},
				});
			}
		},
	},
});
</script>

<style scoped>
.lane {
	background: var(--evcc-gray-10);
}
.chips {
	/* allow shrinking below content width so chip titles can truncate */
	min-width: 0;
}
.lane-num {
	/* constant gutter, wide enough for two digits */
	min-width: 2ch;
}
</style>
