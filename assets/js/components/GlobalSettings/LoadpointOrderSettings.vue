<template>
	<div class="loadpoint-order-settings">
		<div class="small text-muted mb-2">
			{{ $t("settings.loadpoints.help") }}
		</div>
		<div class="loadpoint-list">
			<div
				v-for="(item, index) in displayList"
				:key="item.index"
				class="loadpoint-item d-flex align-items-center p-2 mb-2 border rounded"
				:class="{
					'loadpoint-item--hidden': !item.visible,
					'loadpoint-item--dragging': draggedIndex === index,
				}"
				draggable="true"
				@dragstart="onDragStart(index, $event)"
				@dragover.prevent
				@drop="onDrop(index, $event)"
				@touchstart="onTouchStart(index, $event)"
				@touchmove="onTouchMove($event)"
				@touchend="onTouchEnd($event)"
			>
				<div class="flex-grow-1">
					{{ item.title }}
				</div>
				<div class="form-check form-switch">
					<input
						:id="`loadpoint-visible-${item.index}`"
						v-model="item.visible"
						class="form-check-input"
						type="checkbox"
						role="switch"
						@change="updateVisibility(item.index, item.visible)"
					/>
					<label
						:for="`loadpoint-visible-${item.index}`"
						class="form-check-label visually-hidden"
					>
						{{
							item.visible
								? $t("settings.loadpoints.visible")
								: $t("settings.loadpoints.hidden")
						}}
					</label>
				</div>
			</div>
		</div>
		<div v-if="displayList.length === 0" class="text-muted text-center py-3">
			{{ $t("main.loadpoint.fallbackName") }}
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { LoadpointCompact } from "@/types/evcc";
import {
	getLoadpointDisplayList,
	setLoadpointOrder,
	setLoadpointVisibility,
	type LoadpointDisplayItem,
} from "@/loadpoint-settings";

export default defineComponent({
	name: "LoadpointOrderSettings",
	props: {
		loadpoints: {
			type: Array as PropType<LoadpointCompact[]>,
			default: () => [],
		},
	},
	data() {
		return {
			draggedIndex: -1,
			touchStartY: 0,
			touchStartTime: 0,
			isDragging: false,
			placeholder: null as HTMLElement | null,
		};
	},
	computed: {
		displayList(): LoadpointDisplayItem[] {
			return getLoadpointDisplayList(this.loadpoints);
		},
	},
	methods: {
		onDragStart(index: number, event: DragEvent) {
			this.draggedIndex = index;
			if (event.dataTransfer) {
				event.dataTransfer.effectAllowed = "move";
				event.dataTransfer.setData("text/plain", index.toString());
			}
		},
		onDrop(targetIndex: number, event: DragEvent) {
			event.preventDefault();
			if (this.draggedIndex === -1 || this.draggedIndex === targetIndex) {
				return;
			}

			const newOrder = [...this.displayList.map((item) => item.index)];
			const draggedItem = newOrder.splice(this.draggedIndex, 1)[0];
			if (draggedItem !== undefined) {
				newOrder.splice(targetIndex, 0, draggedItem);
				setLoadpointOrder(newOrder);
			}
			this.draggedIndex = -1;
		},
		updateVisibility(loadpointIndex: number, visible: boolean) {
			setLoadpointVisibility(loadpointIndex, visible);
		},
		onTouchStart(index: number, event: TouchEvent) {
			this.draggedIndex = index;
			this.touchStartY = event.touches[0]?.clientY || 0;
			this.touchStartTime = Date.now();
			this.isDragging = false;
		},
		onTouchMove(event: TouchEvent) {
			if (this.draggedIndex === -1) return;

			event.preventDefault();
			const touch = event.touches[0];
			if (!touch) return;

			const moveDistance = Math.abs(touch.clientY - this.touchStartY);
			const moveTime = Date.now() - this.touchStartTime;

			// Start dragging if moved more than 10px or after 200ms
			if (!this.isDragging && (moveDistance > 10 || moveTime > 200)) {
				this.isDragging = true;
			}

			if (this.isDragging) {
				// Find the element under the touch point
				const elementBelow = document.elementFromPoint(touch.clientX, touch.clientY);
				const targetItem = elementBelow?.closest(".loadpoint-item");

				if (targetItem && this.$el) {
					const items = Array.from(this.$el.querySelectorAll(".loadpoint-item"));
					const targetIndex = items.indexOf(targetItem);

					if (targetIndex !== -1 && targetIndex !== this.draggedIndex) {
						this.reorderItems(targetIndex);
					}
				}
			}
		},
		onTouchEnd(event: TouchEvent) {
			if (this.draggedIndex === -1) return;

			// If we weren't dragging, treat it as a tap (don't prevent default)
			if (!this.isDragging) {
				this.draggedIndex = -1;
				return;
			}

			event.preventDefault();
			this.draggedIndex = -1;
			this.isDragging = false;
		},
		reorderItems(targetIndex: number) {
			if (this.draggedIndex === -1 || this.draggedIndex === targetIndex) {
				return;
			}

			const newOrder = [...this.displayList.map((item) => item.index)];
			const draggedItem = newOrder.splice(this.draggedIndex, 1)[0];
			if (draggedItem !== undefined) {
				newOrder.splice(targetIndex, 0, draggedItem);
				setLoadpointOrder(newOrder);
				this.draggedIndex = targetIndex;
			}
		},
	},
});
</script>

<style scoped>
.loadpoint-item {
	cursor: move;
	user-select: none;
	background-color: var(--evcc-box);
	border-color: var(--bs-border-color-translucent) !important;
	transition: all var(--evcc-transition-fast) ease-in-out;
}

.loadpoint-item:hover {
	background-color: var(--bs-secondary-bg);
}

.loadpoint-item--hidden {
	opacity: 0.6;
}

.loadpoint-item--dragging {
	opacity: 0.8;
	transform: scale(1.02);
	box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
	z-index: 10;
}

.form-check-input {
	margin-top: 0;
}

.form-check {
	display: flex;
	align-items: center;
}

.loadpoint-list:empty + .text-muted {
	display: block;
}
</style>
