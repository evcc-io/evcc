<template>
	<header class="d-flex justify-content-between align-items-center py-3 py-md-4">
		<h1 class="mb-1 pt-1 d-flex text-nowrap text-truncate">
			<router-link class="evcc-default-text" to="/" data-testid="home-link">
				<shopicon-regular-home size="s" class="home"></shopicon-regular-home>
			</router-link>
			<div size="s" class="mx-2 flex-grow-0 flex-shrink-0 fw-normal">/</div>
			<span class="text-truncate">{{ title }}</span>
		</h1>
		<TopNavigation v-bind="topNavigation" />
	</header>
</template>

<script>
import "@h2d2/shopicons/es/regular/home";
import TopNavigation from "./TopNavigation.vue";
import collector from "../mixins/collector";
import store from "../store";

export default {
	name: "TopHeader",
	mixins: [collector],
	components: {
		TopNavigation,
	},
	props: {
		title: String,
	},
	computed: {
		topNavigation: function () {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopNavigation, store.state) };
		},
	},
};
</script>

<style scoped>
.home {
	height: 22px;
	width: 22px;
	position: relative;
	top: -3px;
}
</style>
