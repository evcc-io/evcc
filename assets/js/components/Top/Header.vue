<template>
	<header
		class="d-flex justify-content-between align-items-center py-3 py-md-4"
		data-testid="header"
	>
		<h1 class="mb-1 pt-1 d-flex text-nowrap text-truncate">
			<span class="text-truncate">{{ title }}</span>
		</h1>
		<TopNavigationArea ref="navigationArea" :notifications="notifications" />
	</header>
</template>

<script lang="ts">
import TopNavigationArea from "./TopNavigationArea.vue";
import { defineComponent, type PropType } from "vue";
import type { Notification } from "@/types/evcc";

export default defineComponent({
	name: "TopHeader",
	components: {
		TopNavigationArea,
	},
	props: {
		title: String,
		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
	},
	methods: {
		requestAuthProvider(providerId: string) {
			const navigationArea = this.$refs["navigationArea"] as
				| InstanceType<typeof TopNavigationArea>
				| undefined;
			navigationArea?.requestAuthProvider(providerId);
		},
	},
});
</script>
