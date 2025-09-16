<template>
	<div class="loadpoint-order-settings">
		<div class="small text-muted mb-2">
			{{ $t("settings.loadpoints.help") }}
		</div>
		<div class="loadpoint-list">
			<DragDropItem
				v-for="(item, index) in displayList"
				:key="item.index"
				:title="item.title"
				:visible="item.visible"
				:is-dragging="isItemBeingDragged(index)"
				@dragstart="onDragStart(index, $event)"
				@drop="onDrop(index, $event)"
				@touchstart="onTouchStart(index, $event)"
				@touchmove="onTouchMove($event)"
				@touchend="onTouchEnd($event)"
			>
				<template #actions>
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
				</template>
			</DragDropItem>
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
import dragDropMixin from "@/mixins/dragDrop";
import DragDropItem from "@/components/DragDrop/DragDropItem.vue";

export default defineComponent({
	name: "LoadpointOrderSettings",
	components: { DragDropItem },
	mixins: [dragDropMixin],
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
		updateVisibility(loadpointIndex: number, visible: boolean) {
			setLoadpointVisibility(loadpointIndex, visible);
		},
		/**
		 * Implementation of handleReorder required by dragDropMixin
		 */
		handleReorder(fromIndex: number, toIndex: number) {
			const newOrder = [...this.displayList.map((item) => item.index)];
			const draggedItem = newOrder.splice(fromIndex, 1)[0];
			if (draggedItem !== undefined) {
				newOrder.splice(toIndex, 0, draggedItem);
				setLoadpointOrder(newOrder);
			}
		},
	},
});
</script>

<style scoped>
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
