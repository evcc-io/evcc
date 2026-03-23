<template>
	<div>
		<div class="small text-muted mb-2 col-form-label">
			{{ $t("settings.loadpoints.help") }}
		</div>
		<DragDropList :values="loadpoints.map((item) => item.id)" @reorder="handleReorder">
			<DragDropItem
				v-for="item in loadpoints"
				:key="item.id"
				:title="item.title"
				:visible="item.visible"
			>
				<template #actions>
					<div class="form-check form-switch">
						<input
							:id="`loadpoint-visible-${item.id}`"
							v-model="item.visible"
							class="form-check-input"
							type="checkbox"
							role="switch"
							:aria-label="getVisibilityLabel(item)"
							:disabled="isLastVisible(item)"
							@change="updateVisibility(item.id, item.visible)"
						/>
					</div>
				</template>
			</DragDropItem>
		</DragDropList>
		<div class="mt-2 text-end">
			<button
				type="button"
				class="btn btn-link btn-sm text-muted"
				:disabled="resetDisabled"
				@click="resetOrder"
			>
				{{ $t("config.general.reset") }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { UiLoadpoint } from "@/types/evcc";
import {
	setLoadpointOrder,
	setLoadpointVisibility,
	resetLoadpointsOrder,
	resetLoadpointsVisible,
} from "@/uiLoadpoints";
import DragDropList from "@/components/Helper/DragDropList.vue";
import DragDropItem from "@/components/Helper/DragDropItem.vue";

export default defineComponent({
	name: "LoadpointOrderSettings",
	components: { DragDropList, DragDropItem },
	props: {
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
	},
	computed: {
		resetDisabled() {
			const allVisible = this.loadpoints.every((lp) => lp.visible);
			const noOrder = this.loadpoints.every((lp) => lp.order === null);
			return allVisible && noOrder;
		},
		visibleCount() {
			return this.loadpoints.filter((lp) => lp.visible).length;
		},
	},
	methods: {
		isLastVisible(item: UiLoadpoint) {
			return item.visible && this.visibleCount <= 1;
		},
		getVisibilityLabel(item: UiLoadpoint) {
			const action = item.visible ? "hide" : "show";
			return this.$t(`settings.loadpoints.${action}`, { title: item.title });
		},
		updateVisibility(loadpointId: string, visible: boolean) {
			setLoadpointVisibility(loadpointId, visible);
		},
		handleReorder(newOrder: string[]) {
			setLoadpointOrder(newOrder);
		},
		resetOrder() {
			resetLoadpointsOrder();
			resetLoadpointsVisible();
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
</style>
