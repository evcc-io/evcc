<template>
	<div class="border rounded-4 px-3">
		<div
			v-for="(item, i) in items"
			:key="item.id"
			class="d-flex align-items-center gap-3 py-2 entry"
			:class="{ 'border-top': i > 0 }"
		>
			<span class="fw-bold text-truncate flex-grow-1">{{ item.title }}</span>
			<button
				v-if="item.add"
				type="button"
				class="btn btn-sm btn-outline-primary btn-pill px-3"
				@click="$emit('add', item.id)"
			>
				{{ $t("config.general.add") }}
			</button>
			<template v-else>
				<span v-if="item.value" class="fw-bold text-nowrap">{{ item.value }}</span>
				<DeviceCardEditIcon editable @edit="$emit('edit', item.id)" />
			</template>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";

export interface DeviceListItem {
	id: number | string;
	title: string;
	// formatted live reading, shown next to the edit icon
	value?: string;
	// not configured yet: add button instead of value and edit icon
	add?: boolean;
}

// compact list of devices: title, optional value and an edit icon, or an add button for missing ones
export default defineComponent({
	name: "DeviceList",
	components: { DeviceCardEditIcon },
	props: {
		items: { type: Array as PropType<DeviceListItem[]>, required: true },
	},
	emits: ["add", "edit"],
});
</script>

<style scoped>
/* edit icon and add button differ in height, rows stay the same */
.entry {
	min-height: 3.5rem;
}
</style>
