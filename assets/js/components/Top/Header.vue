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
		<TopNavigation v-bind="topNavigation" />
	</header>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/settings";
import Navigation from "./Navigation.vue";
import collector from "@/mixins/collector";
import store from "@/store";
import { defineComponent } from "vue";

export default defineComponent({
	name: "TopHeader",
	components: {
		TopNavigation: Navigation,
	},
	mixins: [collector],
	props: {
		showConfig: Boolean,
		title: String,
	},
	computed: {
		topNavigation() {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(Navigation, store.state) };
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
