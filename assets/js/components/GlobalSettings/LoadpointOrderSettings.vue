<template>
	<div>
		<div class="small text-muted mb-2 col-form-label">
			{{ $t("settings.loadpoints.help") }}
		</div>
		<DragDropList :values="displayList.map((item) => item.index)" @reorder="handleReorder">
			<DragDropItem
				v-for="item in displayList"
				:key="item.index"
				:title="item.title"
				:visible="item.visible"
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
		</DragDropList>
		<div v-if="displayList.length > 0" class="mt-2 text-end">
			<button type="button" class="btn btn-link btn-sm text-muted" @click="resetOrder">
				{{ $t("config.general.reset") }}
			</button>
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
	resetLoadpointOrder,
	type LoadpointDisplayItem,
} from "@/loadpoint-display";
import DragDropList from "@/components/Helper/DragDropList.vue";
import DragDropItem from "@/components/Helper/DragDropItem.vue";

export default defineComponent({
	name: "LoadpointOrderSettings",
	components: { DragDropList, DragDropItem },
	props: {
		loadpoints: {
			type: Array as PropType<LoadpointCompact[]>,
			default: () => [],
		},
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
		handleReorder(newOrder: number[]) {
			setLoadpointOrder(newOrder);
		},
		resetOrder() {
			resetLoadpointOrder(this.loadpoints.length);
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
