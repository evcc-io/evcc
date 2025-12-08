<template>
	<header
		class="d-flex justify-content-between align-items-center py-3 py-md-4"
		data-testid="header"
	>
		<h1 class="mb-1 pt-1 d-flex text-nowrap text-truncate">
			<router-link class="evcc-default-text" to="/" data-testid="home-link">
				<shopicon-regular-home size="s" class="icon"></shopicon-regular-home>
			</router-link>
			<div v-if="showConfig" class="d-flex">
				<div size="s" class="mx-2 flex-grow-0 flex-shrink-0 fw-normal">/</div>
				<router-link to="/config" class="evcc-default-text text-decoration-none">
					<shopicon-regular-settings
						size="s"
						class="icon d-block d-sm-none"
					></shopicon-regular-settings>
					<span class="d-none d-sm-block">{{ $t("config.main.title") }}</span>
				</router-link>
			</div>
			<div size="s" class="mx-2 flex-grow-0 flex-shrink-0 fw-normal">/</div>
			<span class="text-truncate">{{ title }}</span>
		</h1>
		<TopNavigationArea ref="navigationArea" :notifications="notifications" />
	</header>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/settings";
import TopNavigationArea from "./TopNavigationArea.vue";
import { defineComponent, type PropType } from "vue";
import type { Notification } from "@/types/evcc";

export default defineComponent({
	name: "TopHeader",
	components: {
		TopNavigationArea,
	},
	props: {
		showConfig: Boolean,
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

<style scoped>
.icon {
	height: 22px;
	width: 22px;
	position: relative;
	top: -3px;
}
</style>
